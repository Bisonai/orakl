package peer

import (
	"bisonai.com/orakl/node/pkg/db"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type PeerModel struct {
	ID  int64  `db:"id" json:"id"`
	Url string `db:"url" json:"url"`
}

type PeerInsertModel struct {
	Url string `db:"url" json:"url" validate:"required"`
}

func sync(c *fiber.Ctx) error {
	payload := new(PeerInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to parse request")
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Any("payload", payload).Err(err).Msg("Failed to validate request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to validate request")
	}

	h, ok := c.Locals("host").(*host.Host)
	if !ok {
		log.Error().Msg("Failed to get host")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get host")
	}

	isAlive, err := libp2pUtils.IsHostAlive(c.Context(), *h, payload.Url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check peer")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to check peer")
	}
	if !isAlive {
		log.Info().Str("peer", payload.Url).Msg("invalid peer")
		return c.Status(fiber.StatusBadRequest).SendString("invalid peer")
	}

	peers, err := db.QueryRows[PeerModel](c.Context(), GetPeer, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute get query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute get query")
	}

	_, err = db.QueryRow[PeerModel](c.Context(), InsertPeer, map[string]any{
		"url": payload.Url})
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute insert query")
	}

	return c.JSON(peers)
}
