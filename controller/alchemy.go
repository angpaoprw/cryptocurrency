package controller

import "github.com/gofiber/fiber/v2"

// AlchemyCallback godoc
// @Summary Alchemy Webhook Callback
// @Description Handle webhook callbacks from Alchemy
// @Tags Callback
// @Accept json
// @Produce json
// @Success 200 {string} string "Alchemy Callback"
// @Router /callback/alchemy [get]
func (s *Controller) AlchemyCallback(c *fiber.Ctx) error {

	return c.SendString("Alchemy Callback")
}
