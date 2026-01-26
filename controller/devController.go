package controller

import "github.com/gofiber/fiber/v2"

// CheckHealth godoc
// @Summary Health Check
// @Description Check if the server is running and healthy
// @Tags Health
// @Accept json
// @Produce plain
// @Success 200 {string} string "Server is healthy"
// @Router /health [get]
func (s *Controller) CheckHealth(c *fiber.Ctx) error {
	return c.SendString("Server is healthy")
}
