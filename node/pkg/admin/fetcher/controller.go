package fetcher

import (
	"bisonai.com/orakl/node/pkg/admin/utils"
	"github.com/gofiber/fiber/v2"
)

func start(c *fiber.Ctx) error {
	err := utils.SendMessage(c, "fetcher", "start", nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher started")
}

func stop(c *fiber.Ctx) error {
	err := utils.SendMessage(c, "fetcher", "stop", nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher stopped")
}

func refresh(c *fiber.Ctx) error {
	err := utils.SendMessage(c, "fetcher", "refresh", nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher refreshed")
}
