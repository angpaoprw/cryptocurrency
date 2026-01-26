package controller

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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
	zap.L().Info("Alchemy callback received",
		zap.String("body", string(body)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	)

	return c.SendString("Alchemy Callback")
}
