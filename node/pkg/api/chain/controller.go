package chain

import (
	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ChainInsertModel struct {
	Name string `db:"name" json:"name" validate:"required"`
}

type ChainModel struct {
	ChainId *utils.CustomInt64 `db:"chain_id" json:"id"`
	Name    string             `db:"name" json:"name" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ChainInsertModel)

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ChainModel](c, InsertChain, map[string]any{"name": payload.Name})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[ChainModel](c, GetChain, nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ChainModel](c, GetChainByID, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func patchById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ChainInsertModel)

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ChainModel](c, UpdateChain, map[string]any{"name": payload.Name, "id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := utils.QueryRow[ChainModel](c, RemoveChain, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
