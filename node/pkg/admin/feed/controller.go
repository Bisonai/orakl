package feed

import (
	"encoding/json"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

type FeedModel struct {
	Id         *int64          `db:"id" json:"id"`
	Name       string          `db:"name" json:"name"`
	Definition json.RawMessage `db:"definition" json:"definition"`
	AdapterId  *int64          `db:"adapter_id" json:"adapterId"`
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[FeedModel](c.Context(), GetFeed, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed get query: " + err.Error())
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[FeedModel](c.Context(), GetFeedById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func getByConfigId(c *fiber.Ctx) error {
	id := c.Params("id")
	results, err := db.QueryRows[FeedModel](c.Context(), GetFeedsByConfigId, map[string]any{"config_id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed get by adapter id query: " + err.Error())
	}
	return c.JSON(results)
}
