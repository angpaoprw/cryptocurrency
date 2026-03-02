package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/angpaoprw/cryptocurrency/api"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/angpaoprw/cryptocurrency/webhook"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// AlchemyCallback godoc
// @Summary Alchemy Webhook Callback
// @Description Handle webhook callbacks from Alchemy
// @Tags Callback
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid signature"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /callback/alchemy [post]
func (s *Controller) AlchemyCallback(c *fiber.Ctx) error {
	body := c.Body()
	logger.Info("Received Alchemy Callback", zap.String("body", string(body)))

	// Validate webhook signature
	signature := c.Get("X-Alchemy-Signature")
	signingKey := os.Getenv("ALCHEMY_WEBHOOK_SIGNING_KEY")

	if signingKey == "" {
		logger.Error("ALCHEMY_WEBHOOK_SIGNING_KEY not configured")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Webhook signing key not configured",
		})
	}

	if !webhook.ValidateAlchemySignature(body, signature, signingKey) {
		logger.Warn("Invalid webhook signature",
			zap.String("signature", signature),
			zap.String("from_ip", c.IP()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid webhook signature",
		})
	}

	var webhook AlchemyWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		logger.Error("Failed to parse webhook", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook payload",
		})
	}

	for _, activity := range webhook.Event.Activity {
		// Check if transaction already exists
		existingTx, err := s.sql.GetTransactionByEventId(c.Context(), webhook.ID)
		if err == nil && existingTx.ID.String() != "" {
			logger.Info("Transaction already exists", zap.String("event_id", webhook.ID))
			continue
		}

		// Determine transaction type and direction
		toWallet, errTo := s.sql.GetWalletByAddress(c.Context(), activity.ToAddress)
		fromWallet, errFrom := s.sql.GetWalletByAddress(c.Context(), activity.FromAddress)

		var walletID uuid.NullUUID
		var transactionType string
		var isMatched bool

		// Check if transaction is FROM our hot/warm wallet (withdrawal/internal)
		if errFrom == nil && (fromWallet.WalletType == "hot" || fromWallet.WalletType == "warm") {
			walletID = uuid.NullUUID{UUID: fromWallet.ID, Valid: true}
			if errTo == nil {
				transactionType = "internal" // Between our wallets
			} else {
				transactionType = "withdrawal" // To external address
			}
			isMatched = true // Withdrawals are always matched

			logger.Info("Outgoing transaction from our wallet",
				zap.String("from", activity.FromAddress),
				zap.String("to", activity.ToAddress),
				zap.String("type", transactionType),
				zap.String("tx_hash", activity.Hash),
			)

			// Check if this is a withdrawal request waiting for confirmation
			if transactionType == "withdrawal" {
				withdrawalRequest, err := s.sql.GetWithdrawalRequestByTxHash(c.Context(), sql.NullString{
					String: activity.Hash,
					Valid:  true,
				})
				if err == nil && withdrawalRequest.Status == "processing" {
					// Update withdrawal request to completed
					transactionID := uuid.NullUUID{UUID: uuid.New(), Valid: true}
					_, err = s.sql.UpdateWithdrawalRequestStatus(c.Context(), db.UpdateWithdrawalRequestStatusParams{
						ID:            withdrawalRequest.ID,
						Status:        "completed",
						TransactionID: transactionID,
					})
					if err != nil {
						logger.Error("Failed to update withdrawal request to completed",
							zap.Error(err),
							zap.String("request_id", withdrawalRequest.ID.String()),
						)
					} else {
						// Update wallet balance (deduct withdrawn amount)
						if errFrom == nil {
							var balance decimal.Decimal
							if fromWallet.Balance.Valid {
								balance, _ = decimal.NewFromString(fromWallet.Balance.String)
							}
							withdrawnAmount, _ := decimal.NewFromString(withdrawalRequest.RequestedAmount)
							newBalance := balance.Sub(withdrawnAmount)
							_, err = s.sql.UpdateWalletBalance(c.Context(), db.UpdateWalletBalanceParams{
								ID:      fromWallet.ID,
								Balance: sql.NullString{String: newBalance.String(), Valid: true},
							})
							if err != nil {
								logger.Error("Failed to update wallet balance after withdrawal",
									zap.Error(err),
									zap.String("wallet_id", fromWallet.ID.String()),
								)
							}
						}

						logger.Info("Withdrawal confirmed on blockchain",
							zap.String("request_id", withdrawalRequest.ID.String()),
							zap.String("tx_hash", activity.Hash),
						)

						// Send withdrawal completed notification synchronously to operator
						if s.notificationClient != nil {
							s.notificationClient.SendNotification(withdrawalRequest.CustomerID, api.EventWithdrawalCompleted, map[string]interface{}{
								"request_id":     withdrawalRequest.ID.String(),
								"transaction_id": transactionID.UUID.String(),
								"tx_hash":        activity.Hash,
								"to_address":     withdrawalRequest.ToAddress,
								"amount":         withdrawalRequest.RequestedAmount,
								"network":        withdrawalRequest.Network,
								"token":          withdrawalRequest.Token,
								"status":         "completed",
							})
						}
					}
				} else if err != nil {
					logger.Warn("Withdrawal transaction not found in database",
						zap.String("tx_hash", activity.Hash),
						zap.String("from_address", activity.FromAddress),
					)
				}
			}
		} else if errTo == nil {
			// Transaction TO our wallet (deposit)
			walletID = uuid.NullUUID{UUID: toWallet.ID, Valid: true}
			transactionType = "deposit"

			// Only check for deposit request match if it's a hot wallet
			if toWallet.WalletType == "hot" {
				depositRequest, err := s.sql.GetPendingDepositByAddress(c.Context(), activity.ToAddress)
				if err == nil {
					isMatched = true
					logger.Info("Matched deposit request",
						zap.String("request_id", depositRequest.ID.String()),
						zap.String("customer_id", depositRequest.CustomerID),
					)

					// Update deposit request
					var receivedAmount decimal.Decimal
					if depositRequest.ReceivedAmount.Valid {
						receivedAmount, _ = decimal.NewFromString(depositRequest.ReceivedAmount.String)
					}
					receivedAmount = receivedAmount.Add(decimal.NewFromFloat(activity.Value))
					status := "partial"

					if depositRequest.ExpectedAmount.Valid {
						expectedAmt, _ := decimal.NewFromString(depositRequest.ExpectedAmount.String)
						if receivedAmount.GreaterThanOrEqual(expectedAmt) {
							status = "completed"
						}
					} else {
						status = "completed"
					}

					txID := uuid.NullUUID{UUID: uuid.New(), Valid: true}
					_, err = s.sql.UpdateDepositRequestStatus(c.Context(), db.UpdateDepositRequestStatusParams{
						ID:             depositRequest.ID,
						Status:         status,
						ReceivedAmount: sql.NullString{String: receivedAmount.String(), Valid: true},
						TransactionID:  txID,
					})
					if err != nil {
						logger.Error("Failed to update deposit request", zap.Error(err))
					} else {
						logger.Info("Deposit request updated",
							zap.String("request_id", depositRequest.ID.String()),
							zap.String("status", status),
						)

						// Send notification when deposit is completed
						if status == "completed" && s.notificationClient != nil {
							notificationData := map[string]interface{}{
								"request_id":      depositRequest.ID.String(),
								"transaction_id":  txID.UUID.String(),
								"tx_hash":         activity.Hash,
								"deposit_address": depositRequest.AssignedAddress,
								"received_amount": receivedAmount.String(),
								"network":         depositRequest.Network,
								"token":           depositRequest.Token,
								"status":          "completed",
							}
							if depositRequest.RefID.Valid {
								notificationData["ref_id"] = depositRequest.RefID.String
							}
							s.notificationClient.SendNotification(depositRequest.CustomerID, api.EventDepositCompleted, notificationData)
						}
					}
				} else {
					// No matching deposit request - mark as unmatched
					isMatched = false
					logger.Warn("Unmatched deposit to hot wallet",
						zap.String("to_address", activity.ToAddress),
						zap.String("from_address", activity.FromAddress),
						zap.Float64("value", activity.Value),
						zap.String("asset", activity.Asset),
					)
				}
			} else {
				// Warm/Cold wallet deposits are always matched (expected)
				isMatched = true
			}
		} else {
			// Transaction doesn't involve our wallets - skip it
			logger.Warn("Transaction doesn't involve our wallets - skipping",
				zap.String("from", activity.FromAddress),
				zap.String("to", activity.ToAddress),
				zap.String("tx_hash", activity.Hash),
			)
			continue
		}

		// Create transaction
		params := db.CreateTransactionParams{
			WalletID:        walletID,
			WebhookID:       webhook.WebhookID,
			EventID:         webhook.ID,
			Network:         webhook.Event.Network,
			TransactionHash: activity.Hash,
			BlockNumber:     activity.BlockNum,
			FromAddress:     activity.FromAddress,
			ToAddress:       activity.ToAddress,
			Value:           decimal.NewFromFloat(activity.Value).String(),
			Asset:           activity.Asset,
			Category:        activity.Category,
			TransactionType: sql.NullString{String: transactionType, Valid: true},
			Status:          sql.NullString{String: "confirmed", Valid: true},
			IsMatched:       sql.NullBool{Bool: isMatched, Valid: true},
			Confirmations:   sql.NullInt32{Int32: 1, Valid: true},
			ContractAddress: sql.NullString{String: activity.RawContract.Address, Valid: true},
			Decimals:        sql.NullInt32{Int32: int32(activity.RawContract.Decimals), Valid: true},
			RawValue:        sql.NullString{String: activity.RawContract.RawValue, Valid: true},
			BlockTimestamp:  activity.BlockTimestamp,
		}

		log.Printf("body before creating transaction %+v", params)

		transaction, err := s.sql.CreateTransaction(c.Context(), params)
		if err != nil {
			logger.Error("Failed to save transaction", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save transaction",
			})
		}

		// Update wallet balance
		if walletID.Valid {
			if transactionType == "deposit" && errTo == nil {
				var balance decimal.Decimal
				if toWallet.Balance.Valid {
					balance, _ = decimal.NewFromString(toWallet.Balance.String)
				}
				newBalance := balance.Add(decimal.NewFromFloat(activity.Value))
				s.sql.UpdateWalletBalance(c.Context(), db.UpdateWalletBalanceParams{
					ID:      toWallet.ID,
					Balance: sql.NullString{String: newBalance.String(), Valid: true},
				})
			} else if transactionType == "withdrawal" && errFrom == nil {
				var balance decimal.Decimal
				if fromWallet.Balance.Valid {
					balance, _ = decimal.NewFromString(fromWallet.Balance.String)
				}
				newBalance := balance.Sub(decimal.NewFromFloat(activity.Value))
				s.sql.UpdateWalletBalance(c.Context(), db.UpdateWalletBalanceParams{
					ID:      fromWallet.ID,
					Balance: sql.NullString{String: newBalance.String(), Valid: true},
				})
			}
		}

		logger.Info("Transaction saved",
			zap.String("id", transaction.ID.String()),
			zap.String("hash", transaction.TransactionHash),
			zap.String("type", transactionType),
			zap.Bool("matched", isMatched),
		)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Webhook processed successfully",
	})
}

// WebhookStatus godoc
// @Summary Get webhook status and monitoring information
// @Description Returns detailed information about the Alchemy webhook registration status
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Failed to retrieve webhook status"
// @Router /admin/webhooks/status [get]
func (s *Controller) WebhookStatus(c *fiber.Ctx) error {
	webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL")
	notifyAPIKey := os.Getenv("ALCHEMY_NOTIFY_API_KEY")

	if webhookBaseURL == "" || notifyAPIKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Webhook configuration not set",
		})
	}

	alchemyClient := api.NewAlchemyWithNotify("", notifyAPIKey)
	webhookURL := fmt.Sprintf("%s/v1/callback/alchemy", webhookBaseURL)

	// Try to get the webhook (assumes ETH_MAINNET for now, empty addresses since we're just checking status)
	webhookData, err := alchemyClient.GetOrCreateWebhook(webhookURL, "ETH_MAINNET", []string{})
	if err != nil {
		logger.Error("Failed to get webhook", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve webhook: %v", err),
		})
	}

	// Get webhook details including address count
	webhookDetails, err := alchemyClient.GetWebhookDetails(webhookData.ID)
	if err != nil {
		logger.Error("Failed to get webhook details", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve webhook details: %v", err),
		})
	}

	// Get total registrations from database
	totalRegistrations, err := s.sql.CountWebhookRegistrationsByWebhookID(c.Context(), webhookData.ID)
	if err != nil {
		logger.Error("Failed to count webhook registrations", zap.Error(err))
		totalRegistrations = 0
	}

	// Get all active registrations for last activity timestamp
	registrations, err := s.sql.GetWebhookRegistrationsByWebhookID(c.Context(), webhookData.ID)
	var lastRegisteredAt string
	if err == nil && len(registrations) > 0 {
		lastRegisteredAt = registrations[0].RegisteredAt.Format("2006-01-02 15:04:05")
	}

	currentAddressCount := len(webhookDetails.Addresses)
	availableCapacity := WebhookAddressLimit - currentAddressCount
	usagePercentage := float64(currentAddressCount) / float64(WebhookAddressLimit) * 100

	warningMessage := ""
	if currentAddressCount >= WebhookAddressWarnLimit {
		warningMessage = fmt.Sprintf("WARNING: Webhook is at %.1f%% capacity (%d/%d addresses). Approaching limit!", usagePercentage, currentAddressCount, WebhookAddressLimit)
	}

	return c.JSON(fiber.Map{
		"webhook_id":             webhookData.ID,
		"webhook_url":            webhookData.WebhookURL,
		"network":                webhookData.Network,
		"webhook_type":           webhookData.WebhookType,
		"is_active":              webhookDetails.IsActive,
		"current_addresses":      currentAddressCount,
		"address_limit":          WebhookAddressLimit,
		"available_capacity":     availableCapacity,
		"usage_percentage":       fmt.Sprintf("%.2f%%", usagePercentage),
		"registrations_in_db":    totalRegistrations,
		"last_registered_at":     lastRegisteredAt,
		"signing_key_configured": os.Getenv("ALCHEMY_WEBHOOK_SIGNING_KEY") != "",
		"warning":                warningMessage,
		"status":                 "operational",
	})
}
