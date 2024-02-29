package aggregator

import (
	"time"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
)

type AggregatorModel struct {
	Id     *int64 `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Active bool   `db:"active" json:"active"`
}

type AggregatorInsertModel struct {
	Name string `db:"name" json:"name" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(AggregatorInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse body for aggregator insert payload: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate aggregator insert payload: " + err.Error())
	}

	result, err := db.QueryRow[AggregatorModel](c.Context(), InsertAggregator, map[string]any{
		"name": payload.Name,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator insert query: " + err.Error())
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	result, err := db.QueryRows[AggregatorModel](c.Context(), GetAggregator, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator get query: " + err.Error())
	}

	return c.JSON(result)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), GetAggregatorById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), DeleteAggregatorById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator delete by id query: " + err.Error())
	}
	return c.JSON(result)
}

func syncWithAdapter(c *fiber.Ctx) error {
	result, err := db.QueryRows[AggregatorModel](c.Context(), SyncAggregator, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator sync with adapter query: " + err.Error())
	}
	return c.JSON(result)
}

func activate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), ActivateAggregator, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator activate query: " + err.Error())
	}

	err = utils.SendMessage(c, bus.AGGREGATOR, bus.ACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	time.Sleep(10 * time.Millisecond)
	return c.JSON(result)
}

func deactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), DeactivateAggregator, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator deactivate query: " + err.Error())
	}

	err = utils.SendMessage(c, bus.AGGREGATOR, bus.DEACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	time.Sleep(10 * time.Millisecond)
	return c.JSON(result)
}
