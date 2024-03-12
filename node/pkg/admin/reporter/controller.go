package reporter

import (
	"strconv"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
)

func activate(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.REPORTER, bus.ACTIVATE_REPORTER, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start reporter: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start reporter: " + resp.Args["error"].(string))
	}
	return c.SendString("reporter activated: " + strconv.FormatBool(resp.Success))
}

func deactivate(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.REPORTER, bus.DEACTIVATE_REPORTER, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop reporter: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop reporter: " + resp.Args["error"].(string))
	}
	return c.SendString("reporter deactivated: " + strconv.FormatBool(resp.Success))
}

func refresh(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.REPORTER, bus.REFRESH_REPORTER, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh reporter: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh reporter: " + resp.Args["error"].(string))
	}
	return c.SendString("reporter refreshed: " + strconv.FormatBool(resp.Success))
}
