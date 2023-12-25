package aggregate

import (
	"encoding/json"
	"go-api/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
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
		log.Panic(err)
	}
	payload := _payload.Data

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Panic(err)
	}

	row, err := utils.QueryRow[AggregateIdModel](c, InsertAggregate, map[string]any{
		"timestamp":     payload.Timestamp.String(),
		"value":         payload.Value,
		"aggregator_id": payload.AggregatorId})
	if err != nil {
		log.Errorf("failed to query row:" + err.Error())
	}

	key := "latestAggregate:" + payload.AggregatorId.String()
	value, err := json.Marshal(AggregateRedisValueModel{Timestamp: payload.Timestamp, Value: payload.Value})
	if err != nil {
		log.Panic(err)
	}

	err = utils.SetRedis(c, key, string(value))
	if err != nil {
		log.Panic(err)
	}

	result := AggregateModel{AggregateId: row.AggregateId, Timestamp: payload.Timestamp, Value: payload.Value, AggregatorId: payload.AggregatorId}
	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[AggregateModel](c, GetAggregate, nil)
	if err != nil {
		log.Panic(err)
	}
	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AggregateModel](c, GetAggregateById, map[string]any{"id": id})
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)
}

func getLatestByHash(c *fiber.Ctx) error {
	hash := c.Params("hash")
	result, err := utils.QueryRow[AggregateModel](c, GetLatestAggregateByHash, map[string]any{"aggregator_hash": hash})
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)
}

func getLatestById(c *fiber.Ctx) error {
	var result AggregateRedisValueModel
	returnVal := utils.CustomInt64(0)

	result.Value = &returnVal
	result.Timestamp = &utils.CustomDateTime{Time: time.Now()}

	id := c.Params("id")
	key := "latestAggregate:" + id
	rawResult, err := utils.GetRedis(c, key)

	if err != nil {
		// query pgsql if not found in redis
		log.Info("querying from pgsql")
		pgsqlResult, err := utils.QueryRow[AggregateModel](c, GetLatestAggregateById, map[string]any{"aggregator_id": id})
		if err != nil {
			log.Panic(err)
		}
		return c.JSON(pgsqlResult)
	}

	err = json.Unmarshal([]byte(rawResult), &result)
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(AggregateInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Panic(err)
	}

	result, err := utils.QueryRow[AggregateModel](c, UpdateAggregateById, map[string]any{"timestamp": payload.Timestamp.String(), "value": payload.Value, "aggregator_id": payload.AggregatorId, "id": id})
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AggregateModel](c, DeleteAggregateById, map[string]any{"id": id})
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)
}
