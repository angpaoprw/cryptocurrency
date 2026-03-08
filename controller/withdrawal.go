package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/angpaoprw/cryptocurrency/api"
	"github.com/angpaoprw/cryptocurrency/cryptocurrency"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type CreateWithdrawalRequestInput struct {
	CustomerID string          `json:"customer_id"`
	ToAddress  string          `json:"to_address"`
	Network    string          `json:"network"`
	Token      string          `json:"token"`
	Amount     decimal.Decimal `json:"amount"`
	Notes      string          `json:"notes"`
}

type CreateWithdrawalResponse struct {
	RequestID    string          `json:"request_id"`
	Status       string          `json:"status"`
	CustomerID   string          `json:"customer_id"`
	ToAddress    string          `json:"to_address"`
	Network      string          `json:"network"`
	Token        string          `json:"token"`
	Amount       decimal.Decimal `json:"amount"`
	EstimatedFee string          `json:"estimated_fee,omitempty"`
	CreatedAt    string          `json:"created_at"`
}

type GetWithdrawalResponse struct {
	RequestID       string          `json:"request_id"`
	Status          string          `json:"status"`
	CustomerID      string          `json:"customer_id"`
	ToAddress       string          `json:"to_address"`
	Network         string          `json:"network"`
	Token           string          `json:"token"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	FeeAmount       decimal.Decimal `json:"fee_amount"`
	ActualAmount    decimal.Decimal `json:"actual_amount"`
	TransactionID   *string         `json:"transaction_id,omitempty"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	CreatedAt       string          `json:"created_at"`
	CompletedAt     *string         `json:"completed_at,omitempty"`
}

// CreateWithdrawalRequest handles internal withdrawal request creation
// @Summary Create a withdrawal request (internal)
// @Description Create a withdrawal request for a customer - automatically processed
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body CreateWithdrawalRequestInput true "Withdrawal request details"
// @Success 200 {object} CreateWithdrawalResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /internal/withdrawal [post]
func (s *Controller) CreateWithdrawalRequest(c *fiber.Ctx) error {
	var input CreateWithdrawalRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	logger.Info("Request Body", zap.Any("body", input))

	// Validate required fields
	if input.CustomerID == "" || input.ToAddress == "" || input.Network == "" || input.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "customer_id, to_address, network, and token are required",
		})
	}

	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "amount must be greater than 0",
		})
	}

	// Check if customer already has an active withdrawal request
	existingByCustomer, err := s.sql.GetActiveWithdrawalByCustomer(c.Context(), input.CustomerID)
	if err == nil && existingByCustomer.ID != uuid.Nil {
		logger.Warn("Customer already has an active withdrawal request",
			zap.String("customer_id", input.CustomerID),
			zap.String("existing_request_id", existingByCustomer.ID.String()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":               "Customer already has an active withdrawal request. Please wait for the current request to complete.",
			"existing_request_id": existingByCustomer.ID.String(),
		})
	}
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Error checking for existing withdrawal by customer", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate withdrawal request",
		})
	}

	// Check if there's already an active withdrawal to the same address
	existingByAddress, err := s.sql.GetActiveWithdrawalByToAddress(c.Context(), input.ToAddress)
	if err == nil && existingByAddress.ID != uuid.Nil {
		logger.Warn("Active withdrawal already exists for this address",
			zap.String("to_address", input.ToAddress),
			zap.String("existing_request_id", existingByAddress.ID.String()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":               "An active withdrawal to this address already exists. Please wait for it to complete.",
			"existing_request_id": existingByAddress.ID.String(),
		})
	}
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Error checking for existing withdrawal by address", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate withdrawal request",
		})
	}

	// Find hot wallet with sufficient balance
	wallet, err := s.sql.GetWalletWithSufficientBalance(c.Context(), db.GetWalletWithSufficientBalanceParams{
		Blockchain: input.Network,
		Token:      input.Token,
		WalletType: "hot",
		Balance:    sql.NullString{String: input.Amount.String(), Valid: true},
	})
	if err != nil {
		logger.Error("No hot wallet with sufficient balance found",
			zap.String("network", input.Network),
			zap.String("token", input.Token),
			zap.String("requested_amount", input.Amount.String()),
			zap.Error(err),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":            "Insufficient balance in all hot wallets",
			"requested_amount": input.Amount.String(),
		})
	}

	// Parse wallet balance for confirmation
	var balance decimal.Decimal
	if wallet.Balance.Valid {
		balance, err = decimal.NewFromString(wallet.Balance.String)
		if err != nil {
			logger.Error("Failed to parse wallet balance",
				zap.Error(err),
				zap.String("wallet_id", wallet.ID.String()),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to verify wallet balance",
			})
		}
	}

	logger.Info("Selected wallet for withdrawal",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("wallet_address", wallet.Address),
		zap.String("balance", balance.String()),
		zap.String("requested", input.Amount.String()),
	)

	// Create withdrawal request in database
	notes := sql.NullString{
		String: input.Notes,
		Valid:  input.Notes != "",
	}

	withdrawalRequest, err := s.sql.CreateWithdrawalRequest(c.Context(), db.CreateWithdrawalRequestParams{
		CustomerID:      input.CustomerID,
		FromWalletID:    wallet.ID,
		ToAddress:       input.ToAddress,
		Network:         input.Network,
		Token:           input.Token,
		RequestedAmount: input.Amount.String(),
		FeeAmount:       sql.NullString{String: "0", Valid: true},
		ActualAmount:    sql.NullString{String: "0", Valid: true},
		Status:          "pending",
		Notes:           notes,
	})
	if err != nil {
		logger.Error("Failed to create withdrawal request",
			zap.Error(err),
			zap.String("customer_id", input.CustomerID),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create withdrawal request",
		})
	}

	logger.Info("Withdrawal request created",
		zap.String("request_id", withdrawalRequest.ID.String()),
		zap.String("customer_id", input.CustomerID),
		zap.String("amount", input.Amount.String()),
	)

	// // Send notification for withdrawal creation
	// if s.notificationClient != nil {
	// 	s.notificationClient.SendNotification(input.CustomerID, api.EventWithdrawalCreated, map[string]interface{}{
	// 		"request_id": withdrawalRequest.ID.String(),
	// 		"to_address": input.ToAddress,
	// 		"network":    input.Network,
	// 		"token":      input.Token,
	// 		"amount":     input.Amount.String(),
	// 		"status":     "pending",
	// 	})
	// }

	// Process withdrawal synchronously
	txHash, err := s.processWithdrawal(withdrawalRequest.ID, wallet)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process withdrawal",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"request_id":  withdrawalRequest.ID.String(),
		"status":      "processing",
		"customer_id": withdrawalRequest.CustomerID,
		"to_address":  withdrawalRequest.ToAddress,
		"network":     withdrawalRequest.Network,
		"token":       withdrawalRequest.Token,
		"amount":      withdrawalRequest.RequestedAmount,
		"tx_hash":     txHash,
		"created_at":  withdrawalRequest.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"message":     "Withdrawal transaction submitted. Status will be updated via Alchemy webhook.",
	})
}

// GetWithdrawalRequest retrieves withdrawal request status
// @Summary Get withdrawal request status
// @Description Get the current status of a withdrawal request
// @Tags Internal
// @Produce json
// @Param id path string true "Withdrawal Request ID"
// @Success 200 {object} GetWithdrawalResponse
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /internal/withdrawal/{id} [get]
func (s *Controller) GetWithdrawalRequest(c *fiber.Ctx) error {
	idParam := c.Params("id")
	requestID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid withdrawal request ID",
		})
	}

	withdrawalRequest, err := s.sql.GetWithdrawalRequest(c.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Withdrawal request not found",
			})
		}
		logger.Error("Failed to get withdrawal request",
			zap.Error(err),
			zap.String("request_id", idParam),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve withdrawal request",
		})
	}

	response := GetWithdrawalResponse{
		RequestID:       withdrawalRequest.ID.String(),
		Status:          withdrawalRequest.Status,
		CustomerID:      withdrawalRequest.CustomerID,
		ToAddress:       withdrawalRequest.ToAddress,
		Network:         withdrawalRequest.Network,
		Token:           withdrawalRequest.Token,
		RequestedAmount: decimal.RequireFromString(withdrawalRequest.RequestedAmount),
		FeeAmount:       decimal.Zero,
		ActualAmount:    decimal.Zero,
		CreatedAt:       withdrawalRequest.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if withdrawalRequest.FeeAmount.Valid {
		response.FeeAmount = decimal.RequireFromString(withdrawalRequest.FeeAmount.String)
	}

	if withdrawalRequest.ActualAmount.Valid {
		response.ActualAmount = decimal.RequireFromString(withdrawalRequest.ActualAmount.String)
	}

	if withdrawalRequest.TransactionID.Valid {
		txID := withdrawalRequest.TransactionID.UUID.String()
		response.TransactionID = &txID
	}

	if withdrawalRequest.ErrorMessage.Valid {
		response.ErrorMessage = &withdrawalRequest.ErrorMessage.String
	}

	if withdrawalRequest.CompletedAt.Valid {
		completedAt := withdrawalRequest.CompletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		response.CompletedAt = &completedAt
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// processWithdrawal processes the withdrawal request synchronously
// Returns the transaction hash and error if any
func (s *Controller) processWithdrawal(requestID uuid.UUID, wallet db.Wallet) (string, error) {
	ctx := context.Background()

	// Update status to processing
	_, err := s.sql.UpdateWithdrawalRequestStatus(ctx, db.UpdateWithdrawalRequestStatusParams{
		ID:     requestID,
		Status: "processing",
	})
	if err != nil {
		logger.Error("Failed to update withdrawal status to processing",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
		return "", fmt.Errorf("failed to update status: %w", err)
	}

	// Get the withdrawal request details
	withdrawalRequest, err := s.sql.GetWithdrawalRequest(ctx, requestID)
	if err != nil {
		logger.Error("Failed to get withdrawal request for processing",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
		s.markWithdrawalFailed(requestID, "Failed to retrieve withdrawal details", "")
		return "", fmt.Errorf("failed to get withdrawal request: %w", err)
	}

	// Decrypt private key
	privateKey, err := s.decryptPrivateKey(wallet.PrivateKey)
	if err != nil {
		logger.Error("Failed to decrypt private key",
			zap.Error(err),
			zap.String("wallet_id", wallet.ID.String()),
		)
		s.markWithdrawalFailed(requestID, "Failed to access wallet credentials", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Import wallet from private key
	cryptoWallet, err := cryptocurrency.ImportWalletFromPrivateKey(privateKey)
	if err != nil {
		logger.Error("Failed to import wallet",
			zap.Error(err),
			zap.String("wallet_id", wallet.ID.String()),
		)
		s.markWithdrawalFailed(requestID, "Failed to import wallet", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("failed to import wallet: %w", err)
	}

	// Get network configuration
	network, err := s.getNetworkConfig(withdrawalRequest.Network)
	if err != nil {
		logger.Error("Failed to get network configuration",
			zap.Error(err),
			zap.String("network", withdrawalRequest.Network),
		)
		s.markWithdrawalFailed(requestID, fmt.Sprintf("Unsupported network: %s", withdrawalRequest.Network), withdrawalRequest.CustomerID)
		return "", fmt.Errorf("unsupported network: %s", withdrawalRequest.Network)
	}

	// Initialize blockchain client
	client, err := cryptocurrency.NewClient(network)
	if err != nil {
		logger.Error("Failed to initialize blockchain client",
			zap.Error(err),
			zap.String("network", withdrawalRequest.Network),
		)
		s.markWithdrawalFailed(requestID, "Failed to connect to blockchain", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	// Check MATIC balance and estimate gas cost
	isERC20 := withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC"
	maticBalance, estimatedGasCost, err := client.CheckGasAndBalance(wallet.Address, isERC20)
	if err != nil {
		logger.Error("Failed to check gas and balance",
			zap.Error(err),
			zap.String("wallet_address", wallet.Address),
		)
		s.markWithdrawalFailed(requestID, "Failed to check wallet balance", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("failed to check gas and balance: %w", err)
	}

	// Convert to MATIC for logging (18 decimals)
	maticBalanceFloat := new(big.Float).Quo(new(big.Float).SetInt(maticBalance), big.NewFloat(1e18))
	estimatedGasFloat := new(big.Float).Quo(new(big.Float).SetInt(estimatedGasCost), big.NewFloat(1e18))
	maticBalanceStr, _ := maticBalanceFloat.Float64()
	estimatedGasStr, _ := estimatedGasFloat.Float64()

	logger.Info("Gas and balance check",
		zap.String("wallet_address", wallet.Address),
		zap.Float64("matic_balance", maticBalanceStr),
		zap.Float64("estimated_gas_cost", estimatedGasStr),
		zap.String("balance_wei", maticBalance.String()),
		zap.String("gas_cost_wei", estimatedGasCost.String()),
	)

	// Check if wallet has enough MATIC for gas
	if maticBalance.Cmp(estimatedGasCost) < 0 {
		errorMsg := fmt.Sprintf("Insufficient MATIC for gas: have %.6f MATIC, need %.6f MATIC", maticBalanceStr, estimatedGasStr)
		logger.Error(errorMsg,
			zap.String("wallet_address", wallet.Address),
			zap.String("request_id", requestID.String()),
		)
		s.markWithdrawalFailed(requestID, errorMsg, withdrawalRequest.CustomerID)
		return "", fmt.Errorf("%s", errorMsg)
	}

	// Convert amount to wei/smallest unit
	requestedAmount, err := decimal.NewFromString(withdrawalRequest.RequestedAmount)
	if err != nil {
		logger.Error("Failed to parse requested amount",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
		s.markWithdrawalFailed(requestID, "Invalid amount format", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("invalid amount format: %w", err)
	}

	amountFloat, _ := requestedAmount.Float64()

	// Use correct conversion function based on token type
	var amountWei *big.Int
	if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
		// USDT/USDC use 6 decimals
		amountWei = cryptocurrency.USDTToSmallestUnit(amountFloat)
	} else {
		// Native tokens (ETH/MATIC/BNB) use 18 decimals
		amountWei = cryptocurrency.EtherToWei(amountFloat)
	}

	var txHash string
	var txErr error

	// Get nonce before first attempt so we can reuse it for retry
	nonce, nonceErr := client.GetNonce(wallet.Address)
	if nonceErr != nil {
		logger.Error("Failed to get nonce",
			zap.Error(nonceErr),
			zap.String("wallet_address", wallet.Address),
		)
		s.markWithdrawalFailed(requestID, "Failed to get nonce", withdrawalRequest.CustomerID)
		return "", fmt.Errorf("failed to get nonce: %w", nonceErr)
	}

	// Execute transfer based on token type (1.5x gas = 150)
	if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
		txHash, txErr = client.TransferUSDTWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 150, nonce)
	} else {
		txHash, txErr = client.TransferNativeWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 150, nonce)
	}

	// If gas-related error, retry with 2x gas (300 = 3x suggested gas price)
	if txErr != nil && cryptocurrency.IsGasRelatedError(txErr) {
		logger.Warn("Transfer failed due to gas pricing, retrying with 2x gas",
			zap.Error(txErr),
			zap.String("request_id", requestID.String()),
		)

		if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
			txHash, txErr = client.TransferUSDTWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 300, nonce)
		} else {
			txHash, txErr = client.TransferNativeWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 300, nonce)
		}

		if txErr != nil {
			logger.Error("Transfer failed even with 2x gas, cancelling withdrawal",
				zap.Error(txErr),
				zap.String("request_id", requestID.String()),
			)
			s.markWithdrawalCancelled(requestID, fmt.Sprintf("Transfer failed after 2x gas retry: %v", txErr), withdrawalRequest.CustomerID)
			return "", fmt.Errorf("blockchain transfer failed after gas retry: %w", txErr)
		}
	} else if txErr != nil {
		// Non-gas error, fail normally
		logger.Error("Failed to execute blockchain transfer",
			zap.Error(txErr),
			zap.String("request_id", requestID.String()),
		)
		s.markWithdrawalFailed(requestID, fmt.Sprintf("Transfer failed: %v", txErr), withdrawalRequest.CustomerID)
		return "", fmt.Errorf("blockchain transfer failed: %w", txErr)
	}

	logger.Info("Blockchain transfer submitted",
		zap.String("request_id", requestID.String()),
		zap.String("tx_hash", txHash),
	)

	// Update withdrawal request with tx_hash (status remains "processing")
	// Completion will be handled by Alchemy webhook callback when transaction is confirmed
	_, err = s.sql.UpdateWithdrawalRequestStatus(ctx, db.UpdateWithdrawalRequestStatusParams{
		ID:     requestID,
		Status: "processing",
		TxHash: sql.NullString{String: txHash, Valid: true},
	})
	if err != nil {
		logger.Error("Failed to store tx_hash in withdrawal request",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
		// Continue anyway, we have the tx_hash
	}

	logger.Info("Withdrawal transaction submitted, waiting for blockchain confirmation",
		zap.String("request_id", requestID.String()),
		zap.String("tx_hash", txHash),
	)

	// Return the transaction hash to the API caller
	// The Alchemy webhook will complete the withdrawal and send notification
	return txHash, nil
}

// markWithdrawalFailed marks a withdrawal request as failed
func (s *Controller) markWithdrawalFailed(requestID uuid.UUID, errorMessage string, customerID string) {
	ctx := context.Background()

	errMsg := sql.NullString{
		String: errorMessage,
		Valid:  true,
	}

	_, err := s.sql.UpdateWithdrawalRequestStatus(ctx, db.UpdateWithdrawalRequestStatusParams{
		ID:           requestID,
		Status:       "failed",
		ErrorMessage: errMsg,
	})
	if err != nil {
		logger.Error("Failed to update withdrawal status to failed",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
	}

	// Send notification for withdrawal failure
	if s.notificationClient != nil {
		body := map[string]interface{}{
			"request_id":    requestID.String(),
			"status":        "failed",
			"error_message": errorMessage,
		}
		err := s.notificationClient.SendNotification(customerID, api.EventWithdrawalFailed, body)
		if err != nil {
			logger.Warn("Failed to send withdrawal failed notification",
				zap.String("customer_id", customerID),
				zap.String("request_id", requestID.String()),
				zap.Error(err),
			)

			bodyBytes, marshalErr := json.Marshal(body)
			if marshalErr != nil {
				logger.Error("Failed to marshal notification body",
					zap.Any("body", body),
					zap.Error(marshalErr),
				)
			} else {
				createFailedAPIReqParam := db.CreateFailedAPIRequestParams{
					WithdrawalRequestID: uuid.NullUUID{Valid: true, UUID: requestID},
					Body:                bodyBytes,
					Error:               err.Error(),
				}

				if _, createErr := s.sql.CreateFailedAPIRequest(ctx, createFailedAPIReqParam); createErr != nil {
					logger.Error("Failed to create failed API request record",
						zap.String("request_id", requestID.String()),
						zap.Error(createErr),
					)
				}
			}
		}
	}
}

// markWithdrawalCancelled marks a withdrawal request as cancelled and notifies the service to return credit
func (s *Controller) markWithdrawalCancelled(requestID uuid.UUID, reason string, customerID string) {
	ctx := context.Background()

	errMsg := sql.NullString{
		String: reason,
		Valid:  true,
	}

	_, err := s.sql.UpdateWithdrawalRequestStatus(ctx, db.UpdateWithdrawalRequestStatusParams{
		ID:           requestID,
		Status:       "cancelled",
		ErrorMessage: errMsg,
	})
	if err != nil {
		logger.Error("Failed to update withdrawal status to cancelled",
			zap.Error(err),
			zap.String("request_id", requestID.String()),
		)
	}

	// Send notification so service can return credit to customer
	if s.notificationClient != nil {
		body := map[string]interface{}{
			"request_id":    requestID.String(),
			"status":        "cancelled",
			"error_message": reason,
		}
		err := s.notificationClient.SendNotification(customerID, api.EventWithdrawalCancelled, body)
		if err != nil {
			logger.Warn("Failed to send withdrawal cancelled notification",
				zap.String("customer_id", customerID),
				zap.String("request_id", requestID.String()),
				zap.Error(err),
			)

			bodyBytes, marshalErr := json.Marshal(body)
			if marshalErr != nil {
				logger.Error("Failed to marshal notification body",
					zap.Any("body", body),
					zap.Error(marshalErr),
				)
			} else {
				createFailedAPIReqParam := db.CreateFailedAPIRequestParams{
					WithdrawalRequestID: uuid.NullUUID{Valid: true, UUID: requestID},
					Body:                bodyBytes,
					Error:               err.Error(),
				}

				if _, createErr := s.sql.CreateFailedAPIRequest(ctx, createFailedAPIReqParam); createErr != nil {
					logger.Error("Failed to create failed API request record",
						zap.String("request_id", requestID.String()),
						zap.Error(createErr),
					)
				}
			}
		}
	}
}

// CancelWithdrawalRequest cancels a stuck pending withdrawal transaction
// @Summary Cancel a stuck withdrawal
// @Description Cancel a withdrawal transaction stuck in pending state, return credit to customer
// @Tags Admin
// @Produce json
// @Param id path string true "Withdrawal Request ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/withdrawal/{id}/cancel [post]
func (s *Controller) CancelWithdrawalRequest(c *fiber.Ctx) error {
	idParam := c.Params("id")
	requestID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid withdrawal request ID",
		})
	}

	withdrawalRequest, err := s.sql.GetWithdrawalRequest(c.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Withdrawal request not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve withdrawal request",
		})
	}

	// Only allow cancellation of processing withdrawals
	if withdrawalRequest.Status != "processing" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot cancel withdrawal in '%s' status, only 'processing' withdrawals can be cancelled", withdrawalRequest.Status),
		})
	}

	w, err := s.sql.GetWallet(c.Context(), withdrawalRequest.FromWalletID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve wallet",
		})
	}

	network, err := s.getNetworkConfig(withdrawalRequest.Network)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Unsupported network: %s", withdrawalRequest.Network),
		})
	}

	client, err := cryptocurrency.NewClient(network)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to connect to blockchain",
		})
	}

	// Check if the original transaction is still pending
	if withdrawalRequest.TxHash.Valid && withdrawalRequest.TxHash.String != "" {
		_, isPending, txErr := client.GetTransactionByHash(withdrawalRequest.TxHash.String)
		if txErr == nil && !isPending {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Transaction is no longer pending (already confirmed or dropped)",
				"tx_hash": withdrawalRequest.TxHash.String,
			})
		}
	}

	privateKey, err := s.decryptPrivateKey(w.PrivateKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to access wallet credentials",
		})
	}

	cryptoWallet, err := cryptocurrency.ImportWalletFromPrivateKey(privateKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to import wallet",
		})
	}

	var cancelTxHash string

	if withdrawalRequest.TxHash.Valid && withdrawalRequest.TxHash.String != "" {
		// Get original tx nonce so we can replace it
		originalTx, _, txErr := client.GetTransactionByHash(withdrawalRequest.TxHash.String)
		if txErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve original transaction",
			})
		}

		// First try: speed up with 2x gas (same nonce, same destination, same amount)
		requestedAmount, _ := decimal.NewFromString(withdrawalRequest.RequestedAmount)
		amountFloat, _ := requestedAmount.Float64()
		var amountWei *big.Int
		if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
			amountWei = cryptocurrency.USDTToSmallestUnit(amountFloat)
		} else {
			amountWei = cryptocurrency.EtherToWei(amountFloat)
		}

		logger.Info("Attempting to speed up stuck transaction with 2x gas",
			zap.String("request_id", requestID.String()),
			zap.String("original_tx_hash", withdrawalRequest.TxHash.String),
			zap.Uint64("nonce", originalTx.Nonce()),
		)

		var speedUpErr error
		if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
			cancelTxHash, speedUpErr = client.TransferUSDTWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 300, originalTx.Nonce())
		} else {
			cancelTxHash, speedUpErr = client.TransferNativeWithGasMultiplier(cryptoWallet, withdrawalRequest.ToAddress, amountWei, 300, originalTx.Nonce())
		}

		if speedUpErr != nil {
			// Speed up failed, cancel by sending 0-value self-transfer
			logger.Warn("Speed up failed, cancelling transaction instead",
				zap.Error(speedUpErr),
				zap.String("request_id", requestID.String()),
			)

			cancelTxHash, err = client.CancelTransaction(cryptoWallet, originalTx.Nonce())
			if err != nil {
				logger.Error("Failed to cancel transaction",
					zap.Error(err),
					zap.String("request_id", requestID.String()),
				)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to cancel transaction",
					"details": err.Error(),
				})
			}

			// Cancelled — mark as cancelled and return credit
			s.markWithdrawalCancelled(requestID, "Transaction cancelled: stuck due to low gas fee", withdrawalRequest.CustomerID)

			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"request_id":     requestID.String(),
				"status":         "cancelled",
				"action":         "cancelled",
				"cancel_tx_hash": cancelTxHash,
				"message":        "Transaction cancelled. Credit return notification sent.",
			})
		}

		// Speed up succeeded — update tx_hash but keep processing status
		_, updateErr := s.sql.UpdateWithdrawalRequestStatus(c.Context(), db.UpdateWithdrawalRequestStatusParams{
			ID:     requestID,
			Status: "processing",
			TxHash: sql.NullString{String: cancelTxHash, Valid: true},
		})
		if updateErr != nil {
			logger.Error("Failed to update tx_hash after speed up",
				zap.Error(updateErr),
				zap.String("request_id", requestID.String()),
			)
		}

		logger.Info("Transaction sped up with 2x gas",
			zap.String("request_id", requestID.String()),
			zap.String("new_tx_hash", cancelTxHash),
		)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"request_id":  requestID.String(),
			"status":      "processing",
			"action":      "sped_up",
			"old_tx_hash": withdrawalRequest.TxHash.String,
			"new_tx_hash": cancelTxHash,
			"message":     "Transaction replayed with 2x gas. Waiting for confirmation.",
		})
	}

	// No tx_hash — just cancel (mark as cancelled, return credit)
	s.markWithdrawalCancelled(requestID, "Cancelled: no transaction hash found", withdrawalRequest.CustomerID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"request_id": requestID.String(),
		"status":     "cancelled",
		"action":     "cancelled",
		"message":    "Withdrawal cancelled. No blockchain transaction to cancel. Credit return notification sent.",
	})
}

// getNetworkConfig returns the network configuration for a given network name
func (s *Controller) getNetworkConfig(network string) (cryptocurrency.Network, error) {
	switch network {
	case "ethereum", "eth":
		return cryptocurrency.EthereumMainnet, nil
	case "polygon", "matic", "POLYGON_MAINNET":
		return cryptocurrency.PolygonMainnet, nil
	case "bsc", "bnb":
		return cryptocurrency.BSCMainnet, nil
	case "sepolia":
		return cryptocurrency.SepoliaTestnet, nil
	default:
		return cryptocurrency.Network{}, fmt.Errorf("unsupported network: %s", network)
	}
}
