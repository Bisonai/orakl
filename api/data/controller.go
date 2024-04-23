package data

import (
	"fmt"

	"bisonai.com/orakl/api/utils"

	"github.com/gofiber/fiber/v2"
)

type BulkInsertModel struct {
	Data []DataInsertModel `json:"data"`
}

type DataInsertModel struct {
	Timestamp    *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Value        *utils.CustomInt64    `db:"value" json:"value" validate:"required"`
	AggregatorId *utils.CustomInt64    `db:"aggregator_id" json:"aggregatorId" validate:"required"`
	FeedId       *utils.CustomInt64    `db:"feed_id" json:"feedId" validate:"required"`
}

type DataResultModel struct {
	DataId       *utils.CustomInt64    `db:"data_id" json:"id"`
	Timestamp    *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Value        *utils.CustomInt64    `db:"value" json:"value" validate:"required"`
	AggregatorId *utils.CustomInt64    `db:"aggregator_id" json:"aggregatorId" validate:"required"`
	FeedId       *utils.CustomInt64    `db:"feed_id" json:"feedId" validate:"required"`
}

type BulkInsertResultModel struct {
	Count int `json:"count"`
}

func bulkInsert(c *fiber.Ctx) error {
	payload := new(BulkInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	query, err := GenerateBulkInsertQuery(payload.Data)
	if err != nil {
		return err
	}
	err = utils.RawQueryWithoutReturn(c, query, nil)

	if err != nil {
		return err
	}

	countResult := BulkInsertResultModel{Count: len(payload.Data)}

	return c.JSON(countResult)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[DataResultModel](c, GetData, nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[DataResultModel](c, GetDataById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func getByFeedId(c *fiber.Ctx) error {
	if !utils.IsTesting(c) {
		panic(fmt.Errorf("not allowed"))
	}
	id := c.Params("id")
	results, err := utils.QueryRows[DataResultModel](c, GetDataByFeedId, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func deleteById(c *fiber.Ctx) error {
	if !utils.IsTesting(c) {
		panic(fmt.Errorf("not allowed"))
	}
	id := c.Params("id")
	result, err := utils.QueryRow[DataResultModel](c, DeleteDataById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
