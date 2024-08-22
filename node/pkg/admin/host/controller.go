package host

import (
	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func getPeerCount(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.LIBP2P, bus.GET_PEER_COUNT, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to libp2p helper")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get peer count: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to get peer count: " + resp.Args["error"].(string))
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get peer count: " + resp.Args["error"].(string))
	}

	return c.JSON(resp.Args)
}

func sync(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.LIBP2P, bus.LIBP2P_SYNC, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to send message to libp2p helper")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to sync libp2p host: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		log.Error().Str("Player", "Admin").Msg("failed to sync libp2p host")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to sync libp2p host: " + resp.Args["error"].(string))
	}

	return c.SendString("libp2p synced")
}
