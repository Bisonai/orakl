package aggregate

import (
	"encoding/json"

	"bisonai.com/orakl/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AggregateRedisValueModel struct {
	Timestamp *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Value     *utils.CustomInt64    `db:"value" json:"value" validate:"required"`
}

type WrappedInsertModel struct {
	Data AggregateInsertModel `json:"data"`
}

type AggregateInsertModel struct {
	Timestamp    *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Value        *utils.CustomInt64    `db:"value" json:"value" validate:"required"`
	AggregatorId *utils.CustomInt64    `db:"aggregator_id" json:"aggregatorId" validate:"required"`
}

type AggregateModel struct {
	AggregateId  *utils.CustomInt64    `db:"aggregate_id" json:"id"`
	Timestamp    *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Value        *utils.CustomInt64    `db:"value" json:"value" validate:"required"`
	AggregatorId *utils.CustomInt64    `db:"aggregator_id" json:"aggregatorId" validate:"required"`
}

type AggregateIdModel struct {
	AggregateId *utils.CustomInt64 `db:"aggregate_id" json:"id"`
}

func insert(c *fiber.Ctx) error {
	_payload := new(WrappedInsertModel)
	if err := c.BodyParser(_payload); err != nil {
		return err
	}
	payload := _payload.Data

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	row, err := utils.QueryRow[AggregateIdModel](c, InsertAggregate, map[string]any{
		"timestamp":     payload.Timestamp.String(),
		"value":         payload.Value,
		"aggregator_id": payload.AggregatorId})
	if err != nil {
		return err
	}

	key := "latestAggregate:" + payload.AggregatorId.String()
	value, err := json.Marshal(AggregateRedisValueModel{Timestamp: payload.Timestamp, Value: payload.Value})
	if err != nil {
		return err
	}

	err = utils.SetRedis(c, key, string(value))
	if err != nil {
		return err
	}

	result := AggregateModel{AggregateId: row.AggregateId, Timestamp: payload.Timestamp, Value: payload.Value, AggregatorId: payload.AggregatorId}
	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[AggregateModel](c, GetAggregate, nil)
	if err != nil {
		return err
	}
	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AggregateModel](c, GetAggregateById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func getLatestByHash(c *fiber.Ctx) error {
	hash := c.Params("hash")
	result, err := utils.QueryRow[AggregateModel](c, GetLatestAggregateByHash, map[string]any{"aggregator_hash": hash})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func getLatestById(c *fiber.Ctx) error {
	var result AggregateRedisValueModel

	id := c.Params("id")
	key := "latestAggregate:" + id
	rawResult, err := utils.GetRedis(c, key)

	if err != nil {
		pgsqlResult, err := utils.QueryRow[AggregateModel](c, GetLatestAggregateById, map[string]any{"aggregator_id": id})
		if err != nil {
			return err
		}
		return c.JSON(pgsqlResult)
	}

	err = json.Unmarshal([]byte(rawResult), &result)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(AggregateInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[AggregateModel](c, UpdateAggregateById, map[string]any{"timestamp": payload.Timestamp.String(), "value": payload.Value, "aggregator_id": payload.AggregatorId, "id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AggregateModel](c, DeleteAggregateById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
