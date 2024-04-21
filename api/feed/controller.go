package feed

import (
	"fmt"

	"bisonai.com/orakl/api/utils"

	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

type FeedWithoutAdapterIdModel struct {
	Name       string          `db:"name" json:"name"`
	Definition json.RawMessage `db:"definition" json:"definition"`
}

type FeedInsertModel struct {
	Name       string             `db:"name" json:"name"`
	Definition json.RawMessage    `db:"definition" json:"definition"`
	AdapterId  *utils.CustomInt64 `db:"adapter_id" json:"adapterId"`
}

type FeedModel struct {
	FeedId     *utils.CustomInt64 `db:"feed_id" json:"id"`
	Name       string             `db:"name" json:"name"`
	Definition json.RawMessage    `db:"definition" json:"definition"`
	AdapterId  *utils.CustomInt64 `db:"adapter_id" json:"adapterId"`
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[FeedModel](c, GetFeed, nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[FeedModel](c, GetFeedById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func getByAdpaterId(c *fiber.Ctx) error {
	if !utils.IsTesting(c) {
		panic(fmt.Errorf("not allowed"))
	}
	id := c.Params("id")
	results, err := utils.QueryRows[FeedModel](c, GetFeedsByAdapterId, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(results)
}

func removeById(c *fiber.Ctx) error {
	if !utils.IsTesting(c) {
		panic(fmt.Errorf("not allowed"))
	}
	id := c.Params("id")
	result, err := utils.QueryRow[FeedModel](c, DeleteFeedById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
