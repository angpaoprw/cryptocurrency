package server

import (
	"os"

	"github.com/angpaoprw/cryptocurrency/controller"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type App struct {
	App        *fiber.App
	Controller *controller.Controller
	Route      fiber.Router
}

func NewServer(controller *controller.Controller) *App {
	app := fiber.New()

	// Logger middleware - logs all incoming requests
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	app.Use(cors.New())

	// Define routes and handlers here
	main_route := app.Group("/v1")
	new_app := App{
		App:        app,
		Controller: controller,
		Route:      main_route,
	}
	new_app.swaggerSetup()

	return &new_app
}

func (s *App) swaggerSetup() {
	// Swagger endpoint
	s.App.Get("/docs/*", fiberSwagger.WrapHandler)
}

func (s *App) Start() error {
	s.Route.Get("/health", s.Controller.CheckHealth)

	//setting up routes
	s.callbackRoute()
	s.depositRoute()
	s.adminRoute()

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000" // Default port if not specified
	}

	return s.App.Listen(":" + port)
}
