package peer

import (
	"bisonai.com/orakl/node/pkg/db"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type PeerModel struct {
	Id     int64  `db:"id" json:"id"`
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

	result, err := db.QueryRow[PeerModel](c.Context(), UpsertPeer, map[string]any{
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

	return c.JSON(peers)
}
