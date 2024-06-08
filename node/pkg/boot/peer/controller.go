package peer

import (
	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type PeerModel struct {
	ID  int64  `db:"id" json:"id"`
	Url string `db:"url" json:"url"`
}

type PeerInsertModel struct {
	Url string `db:"url" json:"url" validate:"required"`
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
		"url": payload.Url})
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

	h, err := libp2pSetup.NewHost(c.Context(), libp2pSetup.WithHolePunch(), libp2pSetup.WithQuic())
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to make host")
	}
	defer func() {
		closeErr := h.Close()
		if closeErr != nil {
			log.Error().Err(closeErr).Msg("Failed to close host")
		}
	}()

	isAlive, err := libp2pUtils.IsHostAlive(c.Context(), h, payload.Url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check peer")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to check peer")
	}
	if !isAlive {
		log.Info().Str("peer", payload.Url).Msg("invalid peer")
		err = h.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close host")
		}
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
