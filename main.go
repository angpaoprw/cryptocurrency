package main

import (
	"context"
	"log"
	"os"

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

	// Start deposit expiration service (checks every 5 seconds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expirationService := service.NewDepositExpirationService(queries)
	go expirationService.Start(ctx)

	logger.Info("Deposit expiration service started (3 minute timeout, checks every 5 seconds)")

	new_controller := controller.NewController(queries)
	app := server.NewServer(new_controller)
	app.Start()
}
