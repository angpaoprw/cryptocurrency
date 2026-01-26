package main

import (
	"log"
	"os"

	"github.com/angpaoprw/cryptocurrency/controller"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/migration"
	"github.com/angpaoprw/cryptocurrency/server"
	"github.com/bytedance/gopkg/util/logger"
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
}

// @title Cryptocurrency API
// @version 1.0
// @description This is a cryptocurrency wallet and blockchain API server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@cryptocurrency.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:5000
// @BasePath /v1
// @schemes http https
func main() {
	conn := db.Connect()
	// Run database migrations
	if err := migration.RunMigrations(conn); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	queries := db.NewStore(conn)

	new_controller := controller.NewController(queries)
	app := server.NewServer(new_controller)
	app.Start()
}
