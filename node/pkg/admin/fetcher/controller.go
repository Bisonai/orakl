package fetcher

import (
	"strconv"

	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func start(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.START_FETCHER_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start fetcher: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to start fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher started: " + strconv.FormatBool(resp.Success))
}

func stop(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.STOP_FETCHER_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop fetcher: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to stop fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher stopped: " + strconv.FormatBool(resp.Success))
}

func refresh(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.FETCHER, bus.REFRESH_FETCHER_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh fetcher: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to refresh fetcher")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh fetcher: " + resp.Args["error"].(string))
	}
	return c.SendString("fetcher refreshed: " + strconv.FormatBool(resp.Success))
}
