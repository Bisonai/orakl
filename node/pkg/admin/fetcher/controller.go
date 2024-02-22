package fetcher

import (
	"bisonai.com/orakl/node/pkg/admin/utils"
	"github.com/gofiber/fiber/v2"
)

func start(c *fiber.Ctx) error {
	utils.SendMessage(c, "fetcher", "start", nil)
	return c.SendString("fetcher started")
}

func stop(c *fiber.Ctx) error {
	utils.SendMessage(c, "fetcher", "stop", nil)
	return c.SendString("fetcher stopped")
}

func refresh(c *fiber.Ctx) error {
	utils.SendMessage(c, "fetcher", "refresh", nil)
	return c.SendString("fetcher refreshed")
}
