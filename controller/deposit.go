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

	wallet, err := s.sql.GetWalletByNetworkAndToken(c.Context(), db.GetWalletByNetworkAndTokenParams{
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
			"error": "No available wallet for this network and token",
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

	depositRequest, err := s.sql.CreateDepositRequest(c.Context(), db.CreateDepositRequestParams{
		CustomerID:      input.CustomerID,
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

	return c.JSON(CreateDepositResponse{
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
	})
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
	Network           string  `json:"network"`
	Token             string  `json:"token"`
	ExpectedAmount    float64 `json:"expected_amount,omitempty"`
	ExpirationMinutes int     `json:"expiration_minutes,omitempty"`
}

type CreateDepositResponse struct {
	RequestID        string    `json:"request_id"`
	CustomerID       string    `json:"customer_id"`
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
