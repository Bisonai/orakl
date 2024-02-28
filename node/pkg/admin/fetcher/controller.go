package fetcher

import (
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
)

func start(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.START_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start fetcher: " + err.Error())
	}
	return c.SendString("fetcher started")
}

func stop(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.STOP_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop fetcher: " + err.Error())
	}
	return c.SendString("fetcher stopped")
}

func refresh(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.REFRESH_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh fetcher: " + err.Error())
	}
	return c.SendString("fetcher refreshed")
}
