package service

import (
	"bisonai.com/orakl/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ServiceModel struct {
	ServiceId *utils.CustomInt64 `db:"service_id" json:"id"`
	Name      string             `db:"name" json:"name" validate:"required"`
}

type ServiceInsertModel struct {
	Name string `db:"name" json:"name" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ServiceInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ServiceModel](c, InsertService, map[string]any{"name": payload.Name})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[ServiceModel](c, GetService, nil)
	if err != nil {
		return err
	}
	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ServiceModel](c, GetServiceById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ServiceInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ServiceModel](c, UpdateServiceById, map[string]any{"id": id, "name": payload.Name})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ServiceModel](c, DeleteServiceById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
