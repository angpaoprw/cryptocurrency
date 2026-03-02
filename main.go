package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"

	"github.com/angpaoprw/cryptocurrency/api"
	"github.com/angpaoprw/cryptocurrency/controller"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/angpaoprw/cryptocurrency/migration"
	"github.com/angpaoprw/cryptocurrency/server"
	"github.com/angpaoprw/cryptocurrency/service"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "github.com/angpaoprw/cryptocurrency/docs" // This is required for swagger
)

func init() {
	if os.Getenv("ENV") != "production" && os.Getenv("ENV") != "staging" {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: Could not load .env file:", err)
			log.Println("Using system environment variables")
		} else {
			log.Println(".env file loaded successfully")
		}
	}

	// Initialize logger
	logger.InitLogger()
}

// @title Cryptocurrency API
// @version 1.0
// @description This is a cryptocurrency wallet and blockchain API server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@cryptocurrency.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @BasePath /v1
func main() {
	conn := db.Connect()
	// Run database migrations
	if err := migration.RunMigrations(conn); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	err := logger.InitLogger()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	queries := db.NewStore(conn)

	// Initialize notification client
	notificationServiceURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	if notificationServiceURL != "" {
		logger.Info("Notification service configured", zap.String("url", notificationServiceURL))
	} else {
		logger.Warn("NOTIFICATION_SERVICE_URL not set - notifications will be disabled")
	}
	notificationClient := api.NewNotificationClient(notificationServiceURL, logger.GetLogger())

	// Load and validate encryption key
	encryptionKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyHex == "" {
		logger.Fatal("ENCRYPTION_KEY environment variable is required for securing private keys")
	}

	encryptionKey, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		logger.Fatal("Invalid ENCRYPTION_KEY format - must be hex encoded", zap.Error(err))
	}

	if len(encryptionKey) != 32 {
		logger.Fatal("Invalid ENCRYPTION_KEY length - must be 32 bytes (64 hex characters)", zap.Int("length", len(encryptionKey)))
	}

	logger.Info("Encryption key loaded successfully", zap.Int("key_length_bytes", len(encryptionKey)))

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start deposit expiration service (checks every 5 seconds)
	expirationService := service.NewDepositExpirationService(queries, notificationClient)
	go expirationService.Start(ctx)
	logger.Info("Deposit expiration service started (3 minute timeout, checks every 5 seconds)")

	// Start wallet balance sync service (checks every 30 seconds)
	balanceSyncService := service.NewWalletBalanceSyncService(queries)
	go balanceSyncService.Start(ctx)
	logger.Info("Wallet balance sync service started (syncs every 30 seconds)")

	// Start failed API retry service (checks every 5 seconds)
	failedAPIRetryService := service.NewFailedAPIRetryService(queries, notificationClient)
	go failedAPIRetryService.Start(ctx)
	logger.Info("Failed API retry service started (retries every 5 seconds)")

	new_controller := controller.NewController(queries, notificationClient, encryptionKey)
	app := server.NewServer(new_controller)
	app.Start()
}
