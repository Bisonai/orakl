package proxy

import (
	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ProxyModel struct {
	Id       *utils.CustomInt64 `db:"id" json:"id"`
	Protocol string             `db:"protocol" json:"protocol" validate:"required"`
	Host     string             `db:"host" json:"host" validate:"required"`
	Port     *utils.CustomInt32 `db:"port" json:"port" validate:"required"`
	Location *string            `db:"location" json:"location"`
}

type ProxyInsertModel struct {
	Protocol string             `db:"protocol" json:"protocol" validate:"required"`
	Host     string             `db:"host" json:"host" validate:"required"`
	Port     *utils.CustomInt32 `db:"port" json:"port" validate:"required"`
	Location *string            `db:"location" json:"location"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ProxyInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ProxyModel](c, InsertProxy, map[string]any{
		"protocol": payload.Protocol,
		"host":     payload.Host,
		"port":     payload.Port,
		"location": &payload.Location})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[ProxyModel](c, GetProxy, nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ProxyModel](c, GetProxyById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")

	payload := new(ProxyInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ProxyModel](c, UpdateProxyById, map[string]any{
		"id":       id,
		"protocol": payload.Protocol,
		"host":     payload.Host,
		"port":     payload.Port,
		"location": &payload.Location})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ProxyModel](c, DeleteProxyById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
