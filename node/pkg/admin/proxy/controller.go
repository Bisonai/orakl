package proxy

import (
	"bisonai.com/miko/node/pkg/db"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type ProxyModel struct {
	ID       *int32  `db:"id" json:"id"`
	Protocol string  `db:"protocol" json:"protocol"`
	Host     string  `db:"host" json:"host"`
	Port     int     `db:"port" json:"port"`
	Location *string `db:"location" json:"location"`
}

type ProxyInsertModel struct {
	Protocol string  `json:"protocol" validate:"required"`
	Host     string  `json:"host" validate:"required"`
	Port     int     `json:"port" validate:"required"`
	Location *string `json:"location"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ProxyInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Str("payload", string(c.Body())).Msg("failed to parse body for proxy insert payload")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to validate proxy insert payload")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	if payload.Location != nil && *payload.Location == "" {
		payload.Location = nil
	}

	result, err := db.QueryRow[ProxyModel](c.Context(), InsertProxy, map[string]any{
		"protocol": payload.Protocol,
		"host":     payload.Host,
		"port":     payload.Port,
		"location": &payload.Location})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute proxy insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute insert proxy query: " + err.Error())
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[ProxyModel](c.Context(), GetProxies, nil)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute get proxy query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get proxy query: " + err.Error())
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[ProxyModel](c.Context(), GetProxyById, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute get proxy by id query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get proxy by id query: " + err.Error())
	}
	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ProxyInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Str("payload", string(c.Body())).Msg("failed to parse body for proxy update payload")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to validate proxy update payload")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	result, err := db.QueryRow[ProxyModel](c.Context(), UpdateProxyById, map[string]any{
		"id":       id,
		"protocol": payload.Protocol,
		"host":     payload.Host,
		"port":     payload.Port,
		"location": &payload.Location})

	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute update proxy by id query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute update proxy by id query: " + err.Error())
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[ProxyModel](c.Context(), DeleteProxyById, map[string]any{"id": id})
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Msg("failed to execute delete proxy by id query")
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute delete proxy by id query: " + err.Error())
	}
	return c.JSON(result)
}
