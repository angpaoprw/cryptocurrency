package controller

import (
	"github.com/angpaoprw/cryptocurrency/logger"
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
	logger.Info("Received Alchemy Callback", zap.String("body", string(body)))

	return c.SendString("Alchemy Callback")
}
