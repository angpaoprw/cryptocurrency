package controller

import "github.com/gofiber/fiber/v2"

func (s *Controller) AlchemyCallback(c *fiber.Ctx) error {

	return c.SendString("Alchemy Callback")
}
