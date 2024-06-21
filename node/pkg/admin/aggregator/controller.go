package aggregator

import (
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func start(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.START_AGGREGATOR_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to start aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator started")
}

func stop(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.STOP_AGGREGATOR_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to stop aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator stopped")
}

func refresh(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.REFRESH_AGGREGATOR_APP, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to refresh aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator refreshed")
}

func activate(c *fiber.Ctx) error {
	id := c.Params("id")

	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.ACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to activate aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to activate aggregator: " + resp.Args["error"].(string))
	}

	return c.JSON(resp)
}

func deactivate(c *fiber.Ctx) error {
	id := c.Params("id")

	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.DEACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to deactivate aggregator")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to deactivate aggregator: " + resp.Args["error"].(string))
	}

	return c.JSON(resp)
}
