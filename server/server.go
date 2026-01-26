package server

import (
	"os"

	"github.com/angpaoprw/cryptocurrency/controller"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type App struct {
	App        *fiber.App
	Controller *controller.Controller
	Route      *fiber.Router
}

func NewServer(controller *controller.Controller) *App {
	app := fiber.New()
	app.Use(cors.New())

	// Define routes and handlers here
	main_route := app.Group("/v1")
	new_app := App{
		App:        app,
		Controller: controller,
		Route:      &main_route,
	}
	new_app.swaggerSetup()

	return &new_app
}

func (s *App) swaggerSetup() {

}

func (s *App) Start() error {
	//setting up routes
	s.callbackRoute()

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000" // Default port if not specified
	}

	return s.App.Listen(":" + port)
}
