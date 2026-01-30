package controller

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/angpaoprw/cryptocurrency/api"
	"github.com/angpaoprw/cryptocurrency/cryptocurrency"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	WebhookAddressLimit     = 100000 // Alchemy's hard limit
	WebhookAddressWarnLimit = 95000  // Soft limit to warn before hitting hard limit
)

// NetworkInfo represents blockchain network information for API responses
type NetworkInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	ChainID     int    `json:"chain_id"`
	Explorer    string `json:"explorer"`
	Description string `json:"description"`
}

// Helper function to convert DB model to API response
func (s *Controller) getNetworkInfo(networkCode string) *NetworkInfo {
	network, err := s.sql.GetBlockchainNetworkByCode(context.Background(), networkCode)
	if err != nil {
		return nil
	}

	return &NetworkInfo{
		Code:        network.Code,
		Name:        network.Name,
		ChainID:     int(network.ChainID),
		Explorer:    network.Explorer,
		Description: network.Description.String,
	}
}

// Helper function to get token info from database
func (s *Controller) getTokenInfo(tokenCode, networkCode string) *TokenInfo {
	token, err := s.sql.GetTokenByCodeAndNetwork(context.Background(), db.GetTokenByCodeAndNetworkParams{
		Code:        tokenCode,
		NetworkCode: networkCode,
	})
	if err != nil {
		return nil
	}

	contractAddr := ""
	if token.ContractAddress.Valid {
		contractAddr = token.ContractAddress.String
	}

	return &TokenInfo{
		Code:            token.Code,
		Name:            token.Name,
		Symbol:          token.Symbol,
		Decimals:        int(token.Decimals),
		ContractAddress: contractAddr,
		NetworkCode:     token.NetworkCode,
		TokenType:       token.TokenType,
	}
}

type CreateWalletRequest struct {
	Network         string `json:"network" example:"ETH_MAINNET"`
	Token           string `json:"token" example:"ETH"`
	WalletType      string `json:"wallet_type" example:"hot"`
	RegisterWebhook *bool  `json:"register_webhook,omitempty" example:"true"` // Optional: defaults to true if not provided
}

type ImportWalletRequest struct {
	Address         string `json:"address" example:"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"`
	Network         string `json:"network" example:"ETH_MAINNET"`
	Token           string `json:"token" example:"ETH"`
	WalletType      string `json:"wallet_type" example:"hot"`
	RegisterWebhook *bool  `json:"register_webhook,omitempty" example:"true"` // Optional: defaults to true if not provided
}

type CreateWalletResponse struct {
	ID           string       `json:"id"`
	Address      string       `json:"address"`
	Network      string       `json:"network"`
	NetworkInfo  *NetworkInfo `json:"network_info,omitempty"`
	Token        string       `json:"token"`
	TokenInfo    *TokenInfo   `json:"token_info,omitempty"`
	WalletType   string       `json:"wallet_type"`
	WebhookID    string       `json:"webhook_id,omitempty"`
	RegisteredAt string       `json:"registered_at,omitempty"`
}

type UpdateWalletRequest struct {
	WalletType *string `json:"wallet_type,omitempty" example:"warm"`
	IsActive   *bool   `json:"is_active,omitempty" example:"false"`
}

type WalletResponse struct {
	ID          string       `json:"id"`
	Address     string       `json:"address"`
	Network     string       `json:"network"`
	NetworkInfo *NetworkInfo `json:"network_info,omitempty"`
	Token       string       `json:"token"`
	TokenInfo   *TokenInfo   `json:"token_info,omitempty"`
	WalletType  string       `json:"wallet_type"`
	Balance     string       `json:"balance"`
	IsActive    bool         `json:"is_active"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type ListWalletsResponse struct {
	Wallets []WalletResponse `json:"wallets"`
	Total   int              `json:"total"`
}

type BootstrapWebhooksRequest struct {
	Network string `json:"network" example:"ETH_MAINNET"`
}

type BootstrapWebhooksResponse struct {
	TotalWallets      int      `json:"total_wallets"`
	SuccessfullyAdded int      `json:"successfully_added"`
	AlreadyRegistered int      `json:"already_registered"`
	Failed            int      `json:"failed"`
	WebhookID         string   `json:"webhook_id"`
	FailedAddresses   []string `json:"failed_addresses,omitempty"`
	Message           string   `json:"message"`
}

// CreateWallet godoc
// @Summary Create a new wallet with optional webhook registration
// @Description Creates a new wallet and optionally registers it with Alchemy webhook. Set register_webhook to false to skip webhook registration.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body CreateWalletRequest true "Wallet creation request"
// @Success 201 {object} CreateWalletResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Webhook registration or wallet creation failed"
// @Router /admin/wallets [post]
func (s *Controller) CreateWallet(c *fiber.Ctx) error {
	var req CreateWalletRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate wallet type
	if req.WalletType != "hot" && req.WalletType != "warm" && req.WalletType != "cold" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid wallet_type. Must be 'hot', 'warm', or 'cold'",
		})
	}

	// Validate required fields
	if req.Network == "" || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "network and token are required",
		})
	}

	// Validate network exists in database
	network, err := s.sql.GetBlockchainNetworkByCode(c.Context(), req.Network)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid network '%s'. Supported networks: ETH_MAINNET, ETH_SEPOLIA, POLYGON_MAINNET, POLYGON_AMOY", req.Network),
		})
	}
	if !network.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Network '%s' is not active", req.Network),
		})
	}

	// Validate token exists for this network
	token, err := s.sql.GetTokenByCodeAndNetwork(c.Context(), db.GetTokenByCodeAndNetworkParams{
		Code:        req.Token,
		NetworkCode: req.Network,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid token '%s' for network '%s'", req.Token, req.Network),
		})
	}
	if !token.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Token '%s' is not active", req.Token),
		})
	}

	// Default register_webhook to true if not provided
	registerWebhook := true
	if req.RegisterWebhook != nil {
		registerWebhook = *req.RegisterWebhook
	}

	logger.Info("Creating new wallet",
		zap.String("network", req.Network),
		zap.String("token", req.Token),
		zap.String("wallet_type", req.WalletType),
		zap.Bool("register_webhook", registerWebhook),
	)

	// Step 1: Generate new wallet
	wallet, err := cryptocurrency.CreateWallet()
	if err != nil {
		logger.Error("Failed to generate wallet", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate wallet",
		})
	}

	logger.Info("Wallet generated",
		zap.String("address", wallet.Address),
		zap.String("public_key", wallet.PublicKeyHex()),
	)

	// Step 2: Register with Alchemy webhook (optional - only if register_webhook is true)
	var webhookID string
	var registeredAt string

	if registerWebhook {
		webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL")
		notifyAPIKey := os.Getenv("ALCHEMY_NOTIFY_API_KEY")

		if webhookBaseURL == "" || notifyAPIKey == "" {
			logger.Error("Webhook configuration missing")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Webhook configuration not set. Please configure WEBHOOK_BASE_URL and ALCHEMY_NOTIFY_API_KEY",
			})
		}

		alchemyClient := api.NewAlchemyWithNotify("", notifyAPIKey)
		webhookURL := fmt.Sprintf("%s/v1/callback/alchemy", webhookBaseURL)

		// Get or create the shared webhook
		webhookData, err := alchemyClient.GetOrCreateWebhook(webhookURL, req.Network, []string{wallet.Address})
		if err != nil {
			logger.Error("Failed to get or create webhook", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to initialize webhook: %v", err),
			})
		}

		logger.Info("Webhook found/created",
			zap.String("webhook_id", webhookData.ID),
			zap.String("network", webhookData.Network),
		)

		// Check webhook address limit
		webhookDetails, err := alchemyClient.GetWebhookDetails(webhookData.ID)
		if err != nil {
			logger.Error("Failed to get webhook details", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to check webhook capacity: %v", err),
			})
		}

		currentAddressCount := len(webhookDetails.Addresses)
		logger.Info("Current webhook address count",
			zap.Int("count", currentAddressCount),
			zap.Int("limit", WebhookAddressLimit),
		)

		if currentAddressCount >= WebhookAddressWarnLimit {
			logger.Warn("Webhook approaching address limit",
				zap.Int("current", currentAddressCount),
				zap.Int("limit", WebhookAddressLimit),
			)
		}

		if currentAddressCount >= WebhookAddressLimit {
			logger.Error("Webhook address limit reached",
				zap.Int("current", currentAddressCount),
				zap.Int("limit", WebhookAddressLimit),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Webhook address limit reached (%d/%d). Cannot create new wallet. Please contact administrator.", currentAddressCount, WebhookAddressLimit),
			})
		}

		// Add address to webhook
		err = alchemyClient.AddAddressesToWebhook(webhookData.ID, []string{wallet.Address})
		if err != nil {
			logger.Error("Failed to add address to webhook", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to register wallet address with webhook: %v", err),
			})
		}

		logger.Info("Address added to webhook",
			zap.String("address", wallet.Address),
			zap.String("webhook_id", webhookData.ID),
		)

		webhookID = webhookData.ID
	} else {
		logger.Info("Skipping webhook registration as requested")
	}

	// Step 3: Save wallet to database
	dbWallet, err := s.sql.CreateWallet(c.Context(), db.CreateWalletParams{
		WalletType: req.WalletType,
		Blockchain: req.Network,
		Token:      req.Token,
		Address:    wallet.Address,
		PublicKey:  wallet.PublicKeyHex(),
		PrivateKey: wallet.PrivateKeyHex(),
		Balance:    sql.NullString{String: "0", Valid: true},
		IsActive:   sql.NullBool{Bool: true, Valid: true},
	})
	if err != nil {
		logger.Error("Failed to save wallet to database", zap.Error(err))
		// TODO: Consider removing address from webhook on database failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save wallet to database",
		})
	}

	logger.Info("Wallet saved to database",
		zap.String("wallet_id", dbWallet.ID.String()),
		zap.String("address", dbWallet.Address),
	)

	// Step 4: Save webhook registration record (only if webhook was registered)
	if registerWebhook && webhookID != "" {
		webhookReg, err := s.sql.CreateWebhookRegistration(c.Context(), db.CreateWebhookRegistrationParams{
			WalletID:         dbWallet.ID,
			AlchemyWebhookID: webhookID,
			Address:          wallet.Address,
			Network:          req.Network,
			IsActive:         true,
		})
		if err != nil {
			logger.Error("Failed to save webhook registration", zap.Error(err))
			// Wallet is already saved, but webhook registration record failed
			// This is not critical, but should be logged
		} else {
			registeredAt = webhookReg.RegisteredAt.Format("2006-01-02 15:04:05")
			logger.Info("Webhook registration saved",
				zap.String("registration_id", webhookReg.ID.String()),
				zap.String("webhook_id", webhookReg.AlchemyWebhookID),
			)
		}
	}

	// Get network info for response
	networkInfo := s.getNetworkInfo(req.Network)
	tokenInfo := s.getTokenInfo(req.Token, req.Network)

	return c.Status(fiber.StatusCreated).JSON(CreateWalletResponse{
		ID:           dbWallet.ID.String(),
		Address:      dbWallet.Address,
		Network:      dbWallet.Blockchain,
		NetworkInfo:  networkInfo,
		Token:        dbWallet.Token,
		TokenInfo:    tokenInfo,
		WalletType:   dbWallet.WalletType,
		WebhookID:    webhookID,
		RegisteredAt: registeredAt,
	})
}

// ImportWallet godoc
// @Summary Import an existing wallet address with optional webhook registration
// @Description Imports an existing wallet address into the database and optionally registers it with Alchemy webhook. Use this for addresses that already exist on the blockchain.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body ImportWalletRequest true "Import wallet request"
// @Success 201 {object} CreateWalletResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Webhook registration or wallet import failed"
// @Router /admin/wallets/import [post]
func (s *Controller) ImportWallet(c *fiber.Ctx) error {
	var req ImportWalletRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "address is required",
		})
	}

	if req.Network == "" || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "network and token are required",
		})
	}

	// Validate network exists in database
	network, err := s.sql.GetBlockchainNetworkByCode(c.Context(), req.Network)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid network '%s'. Supported networks: ETH_MAINNET, ETH_SEPOLIA, POLYGON_MAINNET, POLYGON_AMOY", req.Network),
		})
	}
	if !network.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Network '%s' is not active", req.Network),
		})
	}

	// Validate token exists for this network
	token, err := s.sql.GetTokenByCodeAndNetwork(c.Context(), db.GetTokenByCodeAndNetworkParams{
		Code:        req.Token,
		NetworkCode: req.Network,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid token '%s' for network '%s'", req.Token, req.Network),
		})
	}
	if !token.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Token '%s' is not active", req.Token),
		})
	}

	// Validate wallet type
	if req.WalletType != "hot" && req.WalletType != "warm" && req.WalletType != "cold" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid wallet_type. Must be 'hot', 'warm', or 'cold'",
		})
	}

	// Default register_webhook to true if not provided
	registerWebhook := true
	if req.RegisterWebhook != nil {
		registerWebhook = *req.RegisterWebhook
	}

	logger.Info("Importing existing wallet",
		zap.String("address", req.Address),
		zap.String("network", req.Network),
		zap.String("token", req.Token),
		zap.String("wallet_type", req.WalletType),
		zap.Bool("register_webhook", registerWebhook),
	)

	// Check if wallet already exists
	existingWallet, err := s.sql.GetWalletByAddress(context.Background(), req.Address)
	if err == nil && existingWallet.ID != uuid.Nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":     "Wallet address already exists in database",
			"wallet_id": existingWallet.ID.String(),
		})
	}

	// Variables to store result
	var dbWallet db.Wallet
	var webhookID string
	var registeredAt string

	// Execute import in a transaction
	err = s.sql.ExecTx(c.Context(), func(q *db.Queries) error {
		// Step 1: Save wallet to database
		wallet, err := q.CreateWallet(c.Context(), db.CreateWalletParams{
			WalletType: req.WalletType,
			Blockchain: req.Network,
			Token:      req.Token,
			Address:    req.Address,
			PublicKey:  "",
			PrivateKey: "",
			Balance:    sql.NullString{String: "0", Valid: true},
			IsActive:   sql.NullBool{Bool: true, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to save wallet: %w", err)
		}

		dbWallet = wallet

		logger.Info("Wallet imported to database",
			zap.String("wallet_id", dbWallet.ID.String()),
			zap.String("address", dbWallet.Address),
		)

		// Step 2: Register with Alchemy webhook (only if register_webhook is true)
		if registerWebhook {
			webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL")
			notifyAPIKey := os.Getenv("ALCHEMY_NOTIFY_API_KEY")

			if webhookBaseURL == "" || notifyAPIKey == "" {
				return fmt.Errorf("webhook configuration not set")
			}

			alchemyClient := api.NewAlchemyWithNotify("", notifyAPIKey)
			webhookURL := fmt.Sprintf("%s/v1/callback/alchemy", webhookBaseURL)

			// Get or create the shared webhook
			webhookData, err := alchemyClient.GetOrCreateWebhook(webhookURL, req.Network, []string{req.Address})
			if err != nil {
				return fmt.Errorf("failed to initialize webhook: %w", err)
			}

			logger.Info("Webhook found/created",
				zap.String("webhook_id", webhookData.ID),
				zap.String("network", webhookData.Network),
			)

			// Check webhook address limit
			webhookDetails, err := alchemyClient.GetWebhookDetails(webhookData.ID)
			if err != nil {
				return fmt.Errorf("failed to check webhook capacity: %w", err)
			}

			currentAddressCount := len(webhookDetails.Addresses)
			logger.Info("Current webhook address count",
				zap.Int("count", currentAddressCount),
				zap.Int("limit", WebhookAddressLimit),
			)

			if currentAddressCount >= WebhookAddressWarnLimit {
				logger.Warn("Webhook approaching address limit",
					zap.Int("current", currentAddressCount),
					zap.Int("warn_limit", WebhookAddressWarnLimit),
					zap.Int("hard_limit", WebhookAddressLimit),
				)
			}

			if currentAddressCount >= WebhookAddressLimit {
				return fmt.Errorf("webhook address limit reached (%d/%d)", currentAddressCount, WebhookAddressLimit)
			}

			// Check if address already exists in webhook (it might have been added during creation)
			addressExists := false
			for _, addr := range webhookDetails.Addresses {
				if addr == req.Address {
					addressExists = true
					break
				}
			}

			// Only add address if it doesn't already exist
			if !addressExists {
				err = alchemyClient.AddAddressesToWebhook(webhookData.ID, []string{req.Address})
				if err != nil {
					return fmt.Errorf("failed to register webhook: %w", err)
				}
				logger.Info("Address added to webhook successfully",
					zap.String("webhook_id", webhookData.ID),
					zap.String("address", req.Address),
				)
			} else {
				logger.Info("Address already exists in webhook",
					zap.String("webhook_id", webhookData.ID),
					zap.String("address", req.Address),
				)
			}

			webhookID = webhookData.ID

			// Save webhook registration record
			webhookReg, err := q.CreateWebhookRegistration(c.Context(), db.CreateWebhookRegistrationParams{
				WalletID:         dbWallet.ID,
				AlchemyWebhookID: webhookID,
				Address:          req.Address,
				Network:          req.Network,
				IsActive:         true,
			})
			if err != nil {
				return fmt.Errorf("failed to save webhook registration: %w", err)
			}

			registeredAt = webhookReg.RegisteredAt.Format("2006-01-02 15:04:05")
			logger.Info("Webhook registration saved",
				zap.String("registration_id", webhookReg.ID.String()),
				zap.String("webhook_id", webhookReg.AlchemyWebhookID),
			)
		}

		return nil
	})

	if err != nil {
		logger.Error("Transaction failed, rolling back", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get network info for response
	networkInfo := s.getNetworkInfo(req.Network)
	tokenInfo := s.getTokenInfo(req.Token, req.Network)

	return c.Status(fiber.StatusCreated).JSON(CreateWalletResponse{
		ID:           dbWallet.ID.String(),
		Address:      dbWallet.Address,
		Network:      dbWallet.Blockchain,
		NetworkInfo:  networkInfo,
		Token:        dbWallet.Token,
		TokenInfo:    tokenInfo,
		WalletType:   dbWallet.WalletType,
		WebhookID:    webhookID,
		RegisteredAt: registeredAt,
	})
}

// BootstrapWebhooks godoc
// @Summary Bootstrap existing wallets to webhook
// @Description Registers all existing hot wallets without webhook registration
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body BootstrapWebhooksRequest true "Bootstrap request"
// @Success 200 {object} BootstrapWebhooksResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Bootstrap failed"
// @Router /admin/webhooks/bootstrap [post]
func (s *Controller) BootstrapWebhooks(c *fiber.Ctx) error {
	var req BootstrapWebhooksRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Network == "" {
		req.Network = "ETH_MAINNET" // Default network
	}

	logger.Info("Starting webhook bootstrap",
		zap.String("network", req.Network),
	)

	// Get webhook configuration
	webhookBaseURL := os.Getenv("WEBHOOK_BASE_URL")
	notifyAPIKey := os.Getenv("ALCHEMY_NOTIFY_API_KEY")

	if webhookBaseURL == "" || notifyAPIKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Webhook configuration not set",
		})
	}

	alchemyClient := api.NewAlchemyWithNotify("", notifyAPIKey)
	webhookURL := fmt.Sprintf("%s/v1/callback/alchemy", webhookBaseURL)

	// Get all hot wallets without webhook registration
	walletsToRegister, err := s.sql.GetWalletsWithoutWebhookRegistration(c.Context(), "hot")
	if err != nil {
		logger.Error("Failed to get wallets without registration", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve wallets",
		})
	}

	totalWallets := len(walletsToRegister)
	logger.Info("Found wallets to register",
		zap.Int("count", totalWallets),
	)

	if totalWallets == 0 {
		// No wallets to register, but we still need to try to get existing webhook
		webhookData, _ := alchemyClient.GetOrCreateWebhook(webhookURL, req.Network, []string{})
		webhookID := ""
		if webhookData != nil {
			webhookID = webhookData.ID
		}
		return c.Status(fiber.StatusOK).JSON(BootstrapWebhooksResponse{
			TotalWallets:      0,
			SuccessfullyAdded: 0,
			Failed:            0,
			WebhookID:         webhookID,
			Message:           "No wallets need registration",
		})
	}

	// Collect addresses to add
	addressesToAdd := make([]string, 0, totalWallets)
	walletMap := make(map[string]db.Wallet)

	for _, wallet := range walletsToRegister {
		addressesToAdd = append(addressesToAdd, wallet.Address)
		walletMap[wallet.Address] = wallet
	}

	// Get or create the shared webhook with first address
	webhookData, err := alchemyClient.GetOrCreateWebhook(webhookURL, req.Network, []string{addressesToAdd[0]})
	if err != nil {
		logger.Error("Failed to get or create webhook", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initialize webhook: %v", err),
		})
	}

	// Check current webhook capacity
	webhookDetails, err := alchemyClient.GetWebhookDetails(webhookData.ID)
	if err != nil {
		logger.Error("Failed to get webhook details", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to check webhook capacity: %v", err),
		})
	}

	currentAddressCount := len(webhookDetails.Addresses)
	availableCapacity := WebhookAddressLimit - currentAddressCount

	logger.Info("Webhook capacity check",
		zap.Int("current", currentAddressCount),
		zap.Int("available", availableCapacity),
		zap.Int("limit", WebhookAddressLimit),
	)

	// Check if we have enough capacity
	if totalWallets > availableCapacity {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Not enough capacity. Need to register %d wallets but only %d slots available", totalWallets, availableCapacity),
		})
	}

	// Add all addresses in batch (skip first address if it was already added during webhook creation)
	addressesToAddBatch := addressesToAdd
	if currentAddressCount == 1 && len(addressesToAdd) > 0 {
		// First address was already added during webhook creation, skip it
		addressesToAddBatch = addressesToAdd[1:]
	}

	if len(addressesToAddBatch) > 0 {
		err = alchemyClient.AddAddressesToWebhook(webhookData.ID, addressesToAddBatch)
		if err != nil {
			logger.Error("Failed to add addresses to webhook", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to register addresses: %v", err),
			})
		}
	}

	logger.Info("Addresses added to webhook",
		zap.Int("count", len(addressesToAdd)),
		zap.String("webhook_id", webhookData.ID),
	)

	// Save webhook registration records
	successCount := 0
	failedCount := 0
	failedAddresses := []string{}

	for address, wallet := range walletMap {
		_, err := s.sql.CreateWebhookRegistration(c.Context(), db.CreateWebhookRegistrationParams{
			WalletID:         wallet.ID,
			AlchemyWebhookID: webhookData.ID,
			Address:          address,
			Network:          req.Network,
			IsActive:         true,
		})
		if err != nil {
			logger.Error("Failed to save webhook registration",
				zap.Error(err),
				zap.String("address", address),
			)
			failedCount++
			failedAddresses = append(failedAddresses, address)
		} else {
			successCount++
		}
	}

	logger.Info("Bootstrap completed",
		zap.Int("total", totalWallets),
		zap.Int("success", successCount),
		zap.Int("failed", failedCount),
	)

	message := fmt.Sprintf("Successfully registered %d wallets with webhook", successCount)
	if failedCount > 0 {
		message += fmt.Sprintf(", but failed to save %d registration records (addresses still added to webhook)", failedCount)
	}

	return c.Status(fiber.StatusOK).JSON(BootstrapWebhooksResponse{
		TotalWallets:      totalWallets,
		SuccessfullyAdded: successCount,
		Failed:            failedCount,
		WebhookID:         webhookData.ID,
		FailedAddresses:   failedAddresses,
		Message:           message,
	})
}

// GetWallet godoc
// @Summary Get wallet by ID
// @Description Retrieves wallet information by wallet ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Wallet ID"
// @Success 200 {object} WalletResponse
// @Failure 404 {object} map[string]interface{} "Wallet not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/wallets/{id} [get]
func (s *Controller) GetWallet(c *fiber.Ctx) error {
	walletID := c.Params("id")
	if walletID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "wallet ID is required",
		})
	}

	walletUUID, err := uuid.Parse(walletID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid wallet ID format",
		})
	}

	wallet, err := s.sql.GetWallet(c.Context(), walletUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "wallet not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve wallet",
		})
	}

	// Get network info
	networkInfo := s.getNetworkInfo(wallet.Blockchain)
	tokenInfo := s.getTokenInfo(wallet.Token, wallet.Blockchain)

	balance := "0"
	if wallet.Balance.Valid {
		balance = wallet.Balance.String
	}

	isActive := false
	if wallet.IsActive.Valid {
		isActive = wallet.IsActive.Bool
	}

	return c.JSON(WalletResponse{
		ID:          wallet.ID.String(),
		Address:     wallet.Address,
		Network:     wallet.Blockchain,
		NetworkInfo: networkInfo,
		Token:       wallet.Token,
		TokenInfo:   tokenInfo,
		WalletType:  wallet.WalletType,
		Balance:     balance,
		IsActive:    isActive,
		CreatedAt:   wallet.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   wallet.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ListWallets godoc
// @Summary List all wallets
// @Description Retrieves a list of all wallets with pagination
// @Tags Admin
// @Accept json
// @Produce json
// @Param limit query int false "Number of results per page" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Param wallet_type query string false "Filter by wallet type (hot, warm, cold)"
// @Success 200 {object} ListWalletsResponse
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/wallets [get]
func (s *Controller) ListWallets(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	walletType := c.Query("wallet_type", "")

	var wallets []db.Wallet
	var err error

	if walletType != "" {
		wallets, err = s.sql.GetWalletsByType(c.Context(), walletType)
	} else {
		wallets, err = s.sql.ListWallets(c.Context(), db.ListWalletsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
	}

	if err != nil {
		logger.Error("Failed to list wallets", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve wallets",
		})
	}

	walletResponses := make([]WalletResponse, 0, len(wallets))
	for _, wallet := range wallets {
		// Get network info
		networkInfo := s.getNetworkInfo(wallet.Blockchain)
		tokenInfo := s.getTokenInfo(wallet.Token, wallet.Blockchain)

		balance := "0"
		if wallet.Balance.Valid {
			balance = wallet.Balance.String
		}

		isActive := false
		if wallet.IsActive.Valid {
			isActive = wallet.IsActive.Bool
		}

		walletResponses = append(walletResponses, WalletResponse{
			ID:          wallet.ID.String(),
			Address:     wallet.Address,
			Network:     wallet.Blockchain,
			NetworkInfo: networkInfo,
			Token:       wallet.Token,
			TokenInfo:   tokenInfo,
			WalletType:  wallet.WalletType,
			Balance:     balance,
			IsActive:    isActive,
			CreatedAt:   wallet.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   wallet.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return c.JSON(ListWalletsResponse{
		Wallets: walletResponses,
		Total:   len(walletResponses),
	})
}

// UpdateWallet godoc
// @Summary Update wallet
// @Description Updates wallet properties like wallet type or active status
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Wallet ID"
// @Param request body UpdateWalletRequest true "Update wallet request"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Wallet not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/wallets/{id} [put]
func (s *Controller) UpdateWallet(c *fiber.Ctx) error {
	walletID := c.Params("id")
	if walletID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "wallet ID is required",
		})
	}

	walletUUID, err := uuid.Parse(walletID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid wallet ID format",
		})
	}

	var req UpdateWalletRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get current wallet
	wallet, err := s.sql.GetWallet(c.Context(), walletUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "wallet not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve wallet",
		})
	}

	// Update wallet type if provided
	if req.WalletType != nil {
		if *req.WalletType != "hot" && *req.WalletType != "warm" && *req.WalletType != "cold" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid wallet_type. Must be 'hot', 'warm', or 'cold'",
			})
		}
		wallet.WalletType = *req.WalletType
	}

	// Update is_active if provided
	if req.IsActive != nil {
		wallet.IsActive = sql.NullBool{Bool: *req.IsActive, Valid: true}
	}

	// Update wallet in database
	updatedWallet, err := s.sql.UpdateWallet(c.Context(), db.UpdateWalletParams{
		ID:         wallet.ID,
		WalletType: wallet.WalletType,
		IsActive:   wallet.IsActive,
	})
	if err != nil {
		logger.Error("Failed to update wallet", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update wallet",
		})
	}

	logger.Info("Wallet updated",
		zap.String("wallet_id", updatedWallet.ID.String()),
		zap.String("wallet_type", updatedWallet.WalletType),
	)

	// Get network info
	networkInfo := s.getNetworkInfo(updatedWallet.Blockchain)
	tokenInfo := s.getTokenInfo(updatedWallet.Token, updatedWallet.Blockchain)

	balance := "0"
	if updatedWallet.Balance.Valid {
		balance = updatedWallet.Balance.String
	}

	isActive := false
	if updatedWallet.IsActive.Valid {
		isActive = updatedWallet.IsActive.Bool
	}

	return c.JSON(WalletResponse{
		ID:          updatedWallet.ID.String(),
		Address:     updatedWallet.Address,
		Network:     updatedWallet.Blockchain,
		NetworkInfo: networkInfo,
		Token:       updatedWallet.Token,
		TokenInfo:   tokenInfo,
		WalletType:  updatedWallet.WalletType,
		Balance:     balance,
		IsActive:    isActive,
		CreatedAt:   updatedWallet.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   updatedWallet.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// GetSupportedNetworks godoc
// @Summary Get supported networks
// @Description Returns a list of supported blockchain networks from database
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} []NetworkInfo
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/networks [get]
func (s *Controller) GetSupportedNetworks(c *fiber.Ctx) error {
	networks, err := s.sql.ListActiveBlockchainNetworks(c.Context())
	if err != nil {
		logger.Error("Failed to list blockchain networks", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve networks",
		})
	}

	networkInfos := make([]NetworkInfo, 0, len(networks))
	for _, network := range networks {
		networkInfos = append(networkInfos, NetworkInfo{
			Code:        network.Code,
			Name:        network.Name,
			ChainID:     int(network.ChainID),
			Explorer:    network.Explorer,
			Description: network.Description.String,
		})
	}

	return c.JSON(networkInfos)
}

// TokenInfo represents token information for API responses
type TokenInfo struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	ContractAddress string `json:"contract_address,omitempty"`
	NetworkCode     string `json:"network_code"`
	TokenType       string `json:"token_type"`
}

// GetTokensByNetwork godoc
// @Summary Get tokens by network
// @Description Returns a list of tokens available on a specific blockchain network
// @Tags Admin
// @Accept json
// @Produce json
// @Param network_code path string true "Network code (e.g., ETH_MAINNET)"
// @Success 200 {object} []TokenInfo
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/networks/{network_code}/tokens [get]
func (s *Controller) GetTokensByNetwork(c *fiber.Ctx) error {
	networkCode := c.Params("network_code")
	if networkCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "network_code is required",
		})
	}

	tokens, err := s.sql.ListTokensByNetwork(c.Context(), networkCode)
	if err != nil {
		logger.Error("Failed to list tokens", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve tokens",
		})
	}

	tokenInfos := make([]TokenInfo, 0, len(tokens))
	for _, token := range tokens {
		contractAddr := ""
		if token.ContractAddress.Valid {
			contractAddr = token.ContractAddress.String
		}

		tokenInfos = append(tokenInfos, TokenInfo{
			Code:            token.Code,
			Name:            token.Name,
			Symbol:          token.Symbol,
			Decimals:        int(token.Decimals),
			ContractAddress: contractAddr,
			NetworkCode:     token.NetworkCode,
			TokenType:       token.TokenType,
		})
	}

	return c.JSON(tokenInfos)
}

// GetAllTokens godoc
// @Summary Get all active tokens
// @Description Returns a list of all active tokens across all networks
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} []TokenInfo
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/tokens [get]
func (s *Controller) GetAllTokens(c *fiber.Ctx) error {
	tokens, err := s.sql.ListAllActiveTokens(c.Context())
	if err != nil {
		logger.Error("Failed to list all tokens", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve tokens",
		})
	}

	tokenInfos := make([]TokenInfo, 0, len(tokens))
	for _, token := range tokens {
		contractAddr := ""
		if token.ContractAddress.Valid {
			contractAddr = token.ContractAddress.String
		}

		tokenInfos = append(tokenInfos, TokenInfo{
			Code:            token.Code,
			Name:            token.Name,
			Symbol:          token.Symbol,
			Decimals:        int(token.Decimals),
			ContractAddress: contractAddr,
			NetworkCode:     token.NetworkCode,
			TokenType:       token.TokenType,
		})
	}

	return c.JSON(tokenInfos)
}
