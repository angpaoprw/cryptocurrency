package controller

import (
	"database/sql"
	"time"

	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// CreateDepositRequest godoc
// @Summary Create a deposit request for a customer
// @Description Assigns a hot wallet to customer for deposit and creates pending request
// @Tags Deposit
// @Accept json
// @Produce json
// @Param request body CreateDepositRequestInput true "Deposit request details"
// @Success 200 {object} CreateDepositResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /deposit/request [post]
func (s *Controller) CreateDepositRequest(c *fiber.Ctx) error {
	var input CreateDepositRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if input.CustomerID == "" || input.Network == "" || input.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "customer_id, network, and token are required",
		})
	}

	// Check if customer already has a pending deposit request
	existingDeposit, err := s.sql.GetPendingDepositByCustomer(c.Context(), input.CustomerID)
	if err == nil && existingDeposit.ID != uuid.Nil {
		logger.Warn("Customer already has a pending deposit request",
			zap.String("customer_id", input.CustomerID),
			zap.String("existing_request_id", existingDeposit.ID.String()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":               "Customer already has a pending deposit request. Please complete or wait for the current request to expire.",
			"existing_request_id": existingDeposit.ID.String(),
		})
	}
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Error checking for existing deposit request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate deposit request",
		})
	}

	// Get available wallet (not currently assigned to another pending deposit)
	wallet, err := s.sql.GetAvailableWalletByNetworkAndToken(c.Context(), db.GetAvailableWalletByNetworkAndTokenParams{
		Blockchain: input.Network,
		Token:      input.Token,
		WalletType: "hot",
	})
	if err != nil {
		logger.Error("No available hot wallet found",
			zap.String("network", input.Network),
			zap.String("token", input.Token),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No available wallet for this network and token. All wallets are currently assigned to active deposits.",
		})
	}

	expirationMinutes := input.ExpirationMinutes
	if expirationMinutes == 0 {
		expirationMinutes = 3
	}

	expiresAt := sql.NullTime{
		Time:  time.Now().Add(time.Duration(expirationMinutes) * time.Minute),
		Valid: true,
	}

	var expectedAmount sql.NullString
	if input.ExpectedAmount > 0 {
		amt := decimal.NewFromFloat(input.ExpectedAmount)
		expectedAmount = sql.NullString{String: amt.String(), Valid: true}
	}

	var refID sql.NullString
	if input.RefID != "" {
		refID = sql.NullString{String: input.RefID, Valid: true}
	}

	depositRequest, err := s.sql.CreateDepositRequest(c.Context(), db.CreateDepositRequestParams{
		CustomerID:      input.CustomerID,
		RefID:           refID,
		WalletID:        wallet.ID,
		AssignedAddress: wallet.Address,
		Network:         input.Network,
		Token:           input.Token,
		ExpectedAmount:  expectedAmount,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		logger.Error("Failed to create deposit request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create deposit request",
		})
	}

	logger.Info("Deposit request created",
		zap.String("request_id", depositRequest.ID.String()),
		zap.String("customer_id", input.CustomerID),
		zap.String("address", depositRequest.AssignedAddress),
	)

	response := CreateDepositResponse{
		RequestID:        depositRequest.ID.String(),
		CustomerID:       depositRequest.CustomerID,
		DepositAddress:   depositRequest.AssignedAddress,
		Network:          depositRequest.Network,
		Token:            depositRequest.Token,
		ExpectedAmount:   input.ExpectedAmount,
		Status:           depositRequest.Status,
		ExpiresAt:        depositRequest.ExpiresAt.Time,
		ExpiresInSeconds: int(time.Until(depositRequest.ExpiresAt.Time).Seconds()),
		CreatedAt:        depositRequest.CreatedAt,
	}
	if depositRequest.RefID.Valid {
		response.RefID = depositRequest.RefID.String
	}
	return c.JSON(response)
}

// GetDepositRequestStatus godoc
// @Summary Get deposit request status
// @Description Check the status of a deposit request
// @Tags Deposit
// @Accept json
// @Produce json
// @Param id path string true "Deposit Request ID"
// @Success 200 {object} DepositRequestStatusResponse
// @Failure 404 {object} map[string]interface{}
// @Router /deposit/request/{id} [get]
func (s *Controller) GetDepositRequestStatus(c *fiber.Ctx) error {
	requestID := c.Params("id")

	id, err := uuid.Parse(requestID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request ID",
		})
	}

	depositRequest, err := s.sql.GetDepositRequest(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Deposit request not found",
		})
	}

	if depositRequest.Status == "pending" || depositRequest.Status == "partial" {
		if depositRequest.ExpiresAt.Valid && depositRequest.ExpiresAt.Time.Before(time.Now()) {
			depositRequest, _ = s.sql.UpdateDepositRequestStatus(c.Context(), db.UpdateDepositRequestStatusParams{
				ID:             depositRequest.ID,
				Status:         "expired",
				ReceivedAmount: depositRequest.ReceivedAmount,
				TransactionID:  depositRequest.TransactionID,
			})
		}
	}

	response := DepositRequestStatusResponse{
		RequestID:      depositRequest.ID.String(),
		CustomerID:     depositRequest.CustomerID,
		DepositAddress: depositRequest.AssignedAddress,
		Network:        depositRequest.Network,
		Token:          depositRequest.Token,
		Status:         depositRequest.Status,
		CreatedAt:      depositRequest.CreatedAt,
	}
	if depositRequest.RefID.Valid {
		response.RefID = depositRequest.RefID.String
	}

	// Convert ReceivedAmount from string
	if depositRequest.ReceivedAmount.Valid {
		amt, _ := decimal.NewFromString(depositRequest.ReceivedAmount.String)
		response.ReceivedAmount = amt.InexactFloat64()
	}

	if depositRequest.ExpectedAmount.Valid {
		amt, _ := decimal.NewFromString(depositRequest.ExpectedAmount.String)
		val := amt.InexactFloat64()
		response.ExpectedAmount = &val
	}

	if depositRequest.ExpiresAt.Valid {
		response.ExpiresAt = &depositRequest.ExpiresAt.Time
	}

	if depositRequest.CompletedAt.Valid {
		response.CompletedAt = &depositRequest.CompletedAt.Time
	}

	if depositRequest.TransactionID.Valid {
		txID := depositRequest.TransactionID.UUID.String()
		response.TransactionID = &txID
	}

	return c.JSON(response)
}

type CreateDepositRequestInput struct {
	CustomerID        string  `json:"customer_id"`
	RefID             string  `json:"ref_id,omitempty"`
	Network           string  `json:"network"`
	Token             string  `json:"token"`
	ExpectedAmount    float64 `json:"expected_amount,omitempty"`
	ExpirationMinutes int     `json:"expiration_minutes,omitempty"`
}

type CreateDepositResponse struct {
	RequestID        string    `json:"request_id"`
	CustomerID       string    `json:"customer_id"`
	RefID            string    `json:"ref_id,omitempty"`
	DepositAddress   string    `json:"deposit_address"`
	Network          string    `json:"network"`
	Token            string    `json:"token"`
	ExpectedAmount   float64   `json:"expected_amount,omitempty"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
	ExpiresInSeconds int       `json:"expires_in_seconds"`
	CreatedAt        time.Time `json:"created_at"`
}

type DepositRequestStatusResponse struct {
	RequestID      string     `json:"request_id"`
	CustomerID     string     `json:"customer_id"`
	RefID          string     `json:"ref_id,omitempty"`
	DepositAddress string     `json:"deposit_address"`
	Network        string     `json:"network"`
	Token          string     `json:"token"`
	ExpectedAmount *float64   `json:"expected_amount,omitempty"`
	ReceivedAmount float64    `json:"received_amount"`
	Status         string     `json:"status"`
	TransactionID  *string    `json:"transaction_id,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CreateInternalDepositRequest handles internal deposit request creation
// @Summary Create a deposit request (internal service)
// @Description Create a deposit request for internal services - sends notification upon creation
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body CreateDepositRequestInput true "Deposit request details"
// @Success 200 {object} CreateDepositResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /internal/deposit [post]
func (s *Controller) CreateInternalDepositRequest(c *fiber.Ctx) error {
	var input CreateDepositRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	logger.Info("Received internal deposit request", zap.Any("input", input))

	if input.CustomerID == "" || input.Network == "" || input.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "customer_id, network, and token are required",
		})
	}

	// Check if customer already has a pending deposit request
	existingDeposit, err := s.sql.GetPendingDepositByCustomer(c.Context(), input.CustomerID)
	if err == nil && existingDeposit.ID != uuid.Nil {
		logger.Warn("Customer already has a pending deposit request",
			zap.String("customer_id", input.CustomerID),
			zap.String("existing_request_id", existingDeposit.ID.String()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":               "Customer already has a pending deposit request. Please complete or wait for the current request to expire.",
			"existing_request_id": existingDeposit.ID.String(),
		})
	}
	if err != nil && err != sql.ErrNoRows {
		logger.Error("Error checking for existing deposit request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate deposit request",
		})
	}

	// Get available wallet (not currently assigned to another pending deposit)
	wallet, err := s.sql.GetAvailableWalletByNetworkAndToken(c.Context(), db.GetAvailableWalletByNetworkAndTokenParams{
		Blockchain: input.Network,
		Token:      input.Token,
		WalletType: "hot",
	})
	if err != nil {
		logger.Error("No available hot wallet found",
			zap.String("network", input.Network),
			zap.String("token", input.Token),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No available wallet for this network and token. All wallets are currently assigned to active deposits.",
		})
	}

	expirationMinutes := input.ExpirationMinutes
	if expirationMinutes == 0 {
		expirationMinutes = 3
	}

	expiresAt := sql.NullTime{
		Time:  time.Now().Add(time.Duration(expirationMinutes) * time.Minute),
		Valid: true,
	}

	var expectedAmount sql.NullString
	if input.ExpectedAmount > 0 {
		amt := decimal.NewFromFloat(input.ExpectedAmount)
		expectedAmount = sql.NullString{String: amt.String(), Valid: true}
	}

	var refID sql.NullString
	if input.RefID != "" {
		refID = sql.NullString{String: input.RefID, Valid: true}
	}

	depositRequest, err := s.sql.CreateDepositRequest(c.Context(), db.CreateDepositRequestParams{
		CustomerID:      input.CustomerID,
		RefID:           refID,
		WalletID:        wallet.ID,
		AssignedAddress: wallet.Address,
		Network:         input.Network,
		Token:           input.Token,
		ExpectedAmount:  expectedAmount,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		logger.Error("Failed to create deposit request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create deposit request",
		})
	}

	logger.Info("Internal deposit request created",
		zap.String("request_id", depositRequest.ID.String()),
		zap.String("customer_id", input.CustomerID),
		zap.String("address", depositRequest.AssignedAddress),
	)

	response := CreateDepositResponse{
		RequestID:        depositRequest.ID.String(),
		CustomerID:       depositRequest.CustomerID,
		RefID:            depositRequest.RefID.String,
		DepositAddress:   depositRequest.AssignedAddress,
		Network:          depositRequest.Network,
		Token:            depositRequest.Token,
		ExpectedAmount:   input.ExpectedAmount,
		Status:           depositRequest.Status,
		ExpiresAt:        depositRequest.ExpiresAt.Time,
		ExpiresInSeconds: int(time.Until(depositRequest.ExpiresAt.Time).Seconds()),
		CreatedAt:        depositRequest.CreatedAt,
	}
	if depositRequest.RefID.Valid {
		response.RefID = depositRequest.RefID.String
	}
	return c.JSON(response)
}

// CancelDepositRequest cancels an active deposit request
// @Summary Cancel a deposit request (internal)
// @Description Cancel a pending or partial deposit request
// @Tags Internal
// @Produce json
// @Param id path string true "Deposit Request ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /internal/deposit/{id} [delete]
func (s *Controller) CancelDepositRequest(c *fiber.Ctx) error {
	idParam := c.Params("id")
	requestID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid deposit request ID",
		})
	}

	// Get current deposit request
	depositRequest, err := s.sql.GetDepositRequest(c.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Deposit request not found",
			})
		}
		logger.Error("Failed to get deposit request",
			zap.Error(err),
			zap.String("request_id", idParam),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve deposit request",
		})
	}

	// Only allow cancellation of pending or partial deposits
	if depositRequest.Status != "pending" && depositRequest.Status != "partial" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "Cannot cancel deposit request",
			"current_status": depositRequest.Status,
			"message":        "Only pending or partial deposits can be cancelled",
		})
	}

	// Cancel the deposit request
	cancelledRequest, err := s.sql.CancelDepositRequest(c.Context(), requestID)
	if err != nil {
		logger.Error("Failed to cancel deposit request",
			zap.Error(err),
			zap.String("request_id", idParam),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to cancel deposit request",
		})
	}

	logger.Info("Deposit request cancelled",
		zap.String("request_id", requestID.String()),
		zap.String("customer_id", depositRequest.CustomerID),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":    "Deposit request cancelled successfully",
		"request_id": cancelledRequest.ID.String(),
		"status":     cancelledRequest.Status,
	})
}
