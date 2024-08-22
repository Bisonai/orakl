package apierr

import (
	"fmt"

	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ErrorInsertModel struct {
	RequestId string                `db:"request_id" json:"requestId" validate:"required"`
	Timestamp *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Code      string                `db:"code" json:"code" validate:"required"`
	Name      string                `db:"name" json:"name" validate:"required"`
	Stack     string                `db:"stack" json:"stack" validate:"required"`
}

type ErrorModel struct {
	ERROR_ID  *utils.CustomInt64    `db:"error_id" json:"id"`
	RequestId string                `db:"request_id" json:"requestId" validate:"required"`
	Timestamp *utils.CustomDateTime `db:"timestamp" json:"timestamp" validate:"required"`
	Code      string                `db:"code" json:"code" validate:"required"`
	Name      string                `db:"name" json:"name" validate:"required"`
	Stack     string                `db:"stack" json:"stack" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ErrorInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ErrorModel](c, InsertError, map[string]any{
		"request_id": payload.RequestId,
		"timestamp":  payload.Timestamp.String(),
		"code":       payload.Code,
		"name":       payload.Name,
		"stack":      payload.Stack})

	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[ErrorModel](c, GetError, nil)
	if err != nil {
		return err
	}
	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ErrorModel](c, GetErrorById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	if !utils.IsTesting(c) {
		panic(fmt.Errorf("not allowed"))
	}
	id := c.Params("id")
	result, err := utils.QueryRow[ErrorModel](c, RemoveErrorById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
