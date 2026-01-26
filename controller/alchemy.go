package controller

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// AlchemyCallback godoc
// @Summary Alchemy Webhook Callback
// @Description Handle webhook callbacks from Alchemy
// @Tags Callback
// @Accept json
// @Produce json
// @Success 200 {string} string "Alchemy Callback"
// @Router /callback/alchemy [get]
func (s *Controller) AlchemyCallback(c *fiber.Ctx) error {
	// Log the incoming request body
	body := c.Body()
	log.Println("Received Alchemy Callback:", string(body))

	return c.SendString("Alchemy Callback")
}
