package fetcher

import (
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
)

func start(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.START_FETCHER, nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher started")
}

func stop(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.STOP_FETCHER, nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher stopped")
}

func refresh(c *fiber.Ctx) error {
	err := utils.SendMessage(c, bus.FETCHER, bus.REFRESH_FETCHER, nil)
	if err != nil {
		panic(err)
	}
	return c.SendString("fetcher refreshed")
}
