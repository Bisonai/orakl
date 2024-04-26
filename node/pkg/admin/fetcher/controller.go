package fetcher

import (
	"strconv"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
)

func start(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.START_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start fetcher: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher started: " + strconv.FormatBool(resp.Success))
}

func stop(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.STOP_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop fetcher: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher stopped: " + strconv.FormatBool(resp.Success))
}

func refresh(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.REFRESH_FETCHER_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh fetcher: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher refreshed: " + strconv.FormatBool(resp.Success))
}

func activate(c *fiber.Ctx) error {
	id := c.Params("id")

	msg, err := utils.SendMessage(c, bus.FETCHER, bus.ACTIVATE_FETCHER, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send activate message to fetcher: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to activate adapter: " + resp.Args["error"].(string))
	}

	return c.JSON(resp)
}

func deactivate(c *fiber.Ctx) error {
	id := c.Params("id")

	msg, err := utils.SendMessage(c, bus.FETCHER, bus.DEACTIVATE_FETCHER, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send deactivate message to fetcher: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to deactivate adapter: " + resp.Args["error"].(string))
	}

	return c.JSON(resp)
}
