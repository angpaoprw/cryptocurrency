package controller

import (
	"database/sql"
	"encoding/json"

	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
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
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /callback/alchemy [post]
func (s *Controller) AlchemyCallback(c *fiber.Ctx) error {
	body := c.Body()
	logger.Info("Received Alchemy Callback", zap.String("body", string(body)))

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
			)
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
