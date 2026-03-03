package controller

import (
	"context"
	"database/sql"
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

	// Execute transfer based on token type
	if withdrawalRequest.Token == "USDT" || withdrawalRequest.Token == "USDC" {
		txHash, txErr = client.TransferUSDT(cryptoWallet, withdrawalRequest.ToAddress, amountWei)
	} else {
		// Native token (ETH, MATIC, BNB)
		txHash, txErr = client.TransferNative(cryptoWallet, withdrawalRequest.ToAddress, amountWei)
	}

	if txErr != nil {
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
	// TODO: Uncomment after running migration 000011 and regenerating SQLC
	// _, err = s.sql.UpdateWithdrawalRequestStatus(ctx, db.UpdateWithdrawalRequestStatusParams{
	// 	ID:     requestID,
	// 	Status: "processing",
	// 	TxHash: sql.NullString{String: txHash, Valid: true},
	// })
	// if err != nil {
	// 	logger.Error("Failed to store tx_hash in withdrawal request",
	// 		zap.Error(err),
	// 		zap.String("request_id", requestID.String()),
	// 	)
	// 	// Continue anyway, we have the tx_hash
	// }

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
		s.notificationClient.SendNotification(customerID, api.EventWithdrawalFailed, map[string]interface{}{
			"request_id":    requestID.String(),
			"status":        "failed",
			"error_message": errorMessage,
		})
	}
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
