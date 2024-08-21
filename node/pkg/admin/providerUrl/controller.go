package providerUrl

import (
	"bisonai.com/miko/node/pkg/db"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type ProviderUrlModel struct {
	ID       *int32 `db:"id" json:"id"`
	ChainId  *int   `db:"chain_id" json:"chain_id"`
	Url      string `db:"url" json:"url"`
	Priority *int   `db:"priority" json:"priority"`
}

type ProviderUrlInsertModel struct {
	ChainId  *int   `db:"chain_id" json:"chain_id" validate:"required"`
	Url      string `db:"url" json:"url" validate:"required"`
	Priority *int   `db:"priority" json:"priority"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ProviderUrlInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Str("payload", string(c.Body())).Msg("failed to parse body for provider insert payload")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse body for provider insert payload: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to validate provider insert payload")
		return c.Status(fiber.StatusBadRequest).SendString("failed to validate provider insert payload: " + err.Error())
	}

	result, err := db.QueryRow[ProviderUrlModel](c.Context(), InsertProviderUrl, map[string]any{
		"chain_id": payload.ChainId,
		"url":      payload.Url,
		"priority": payload.Priority,
	})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute provider insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute provider insert query: " + err.Error())
	}
	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[ProviderUrlModel](c.Context(), GetProviderUrl, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute provider get query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute provider get query: " + err.Error())
	}
	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[ProviderUrlModel](c.Context(), GetProviderUrlById, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute provider get by id query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute provider get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[ProviderUrlModel](c.Context(), DeleteProviderUrlById, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute provider delete by id query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute provider delete by id query: " + err.Error())
	}
	return c.JSON(result)
}
