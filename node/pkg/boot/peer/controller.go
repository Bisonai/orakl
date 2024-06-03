package peer

import (
	"fmt"

	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type PeerModel struct {
	ID     int64  `db:"id" json:"id"`
	Ip     string `db:"ip" json:"ip"`
	Port   int    `db:"port" json:"port"`
	HostId string `db:"host_id" json:"host_id"`
}

type PeerInsertModel struct {
	Ip     string `db:"ip" json:"ip" validate:"required"`
	Port   int    `db:"port" json:"port" validate:"required"`
	HostId string `db:"host_id" json:"host_id" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(PeerInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to parse request")
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Err(err).Msg("Failed to validate request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to validate request")
	}

	result, err := db.QueryRow[PeerModel](c.Context(), InsertPeer, map[string]any{
		"ip":      payload.Ip,
		"port":    payload.Port,
		"host_id": payload.HostId})
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute insert query")
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	result, err := db.QueryRows[PeerModel](c.Context(), GetPeer, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute get query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute get query")
	}
	return c.JSON(result)
}

func sync(c *fiber.Ctx) error {
	payload := new(PeerInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to parse request")
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Err(err).Msg("Failed to validate request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to validate request")
	}

	h, err := libp2pSetup.MakeHost(0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to make host")
	}

	connectionUrl := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", payload.Ip, payload.Port, payload.HostId)
	isAlive, err := libp2pUtils.IsHostAlive(c.Context(), h, connectionUrl)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check peer")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to check peer")
	}
	if !isAlive {
		log.Info().Str("peer", connectionUrl).Msg("invalid peer")
		err = h.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close host")
		}
		log.Info().Str("peer", connectionUrl).Msg("invalid peer")
		return c.Status(fiber.StatusBadRequest).SendString("invalid peer")
	}

	peers, err := db.QueryRows[PeerModel](c.Context(), GetPeer, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute get query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute get query")
	}

	_, err = db.QueryRow[PeerModel](c.Context(), InsertPeer, map[string]any{
		"ip":      payload.Ip,
		"port":    payload.Port,
		"host_id": payload.HostId})
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute insert query")
	}

	err = h.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close host")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to close host")
	}

	return c.JSON(peers)
}
