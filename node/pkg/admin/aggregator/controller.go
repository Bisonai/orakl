package aggregator

import (
	"errors"
	"os"
	"strconv"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	chainutils "bisonai.com/orakl/node/pkg/chain/utils"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
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

func renewSigner(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.RENEW_SIGNER, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to reporter")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh signer: " + err.Error())
	}
	resp := <-msg.Response

	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to refresh signer")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh signer: " + resp.Args["error"].(string))
	}
	return c.SendString("s refreshed: " + strconv.FormatBool(resp.Success))
}

func getSigner(c *fiber.Ctx) error {
	signerpk, err := chainutils.LoadSignerPk(c.Context())
	if err != nil && !errors.Is(err, errorsentinel.ErrChainSignerPKNotFound) {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get signer: " + err.Error())
	}

	if signerpk == "" {
		signerpk := os.Getenv("SIGNER_PK")
		if signerpk == "" {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to get signer, no signer set")
		}
	}

	addr, err := chainutils.StringPkToAddressHex(signerpk)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get signer: " + err.Error())
	}
	return c.JSON(fiber.Map{"signer": addr})
}
