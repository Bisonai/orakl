package vrf

import (
	"bisonai.com/orakl/node/pkg/api/chain"
	"bisonai.com/orakl/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type VrfModel struct {
	VrfKeyId *utils.CustomInt64 `db:"vrf_key_id" json:"id"`
	Sk       string             `db:"sk" json:"sk" validate:"required"`
	Pk       string             `db:"pk" json:"pk" validate:"required"`
	PkX      string             `db:"pk_x" json:"pkX" validate:"required"`
	PkY      string             `db:"pk_y" json:"pkY" validate:"required"`
	KeyHash  string             `db:"key_hash" json:"keyHash" validate:"required"`
	Chain    string             `db:"chain_name" json:"chain" validate:"required"`
}

type VrfUpdateModel struct {
	Sk      string `db:"sk" json:"sk" validate:"required"`
	Pk      string `db:"pk" json:"pk" validate:"required"`
	PkX     string `db:"pk_x" json:"pkX" validate:"required"`
	PkY     string `db:"pk_y" json:"pkY" validate:"required"`
	KeyHash string `db:"key_hash" json:"keyHash" validate:"required"`
}

type VrfInsertModel struct {
	Sk      string `db:"sk" json:"sk" validate:"required"`
	Pk      string `db:"pk" json:"pk" validate:"required"`
	PkX     string `db:"pk_x" json:"pkX" validate:"required"`
	PkY     string `db:"pk_y" json:"pkY" validate:"required"`
	KeyHash string `db:"key_hash" json:"keyHash" validate:"required"`
	Chain   string `db:"chain_name" json:"chain" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(VrfInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": payload.Chain})
	if err != nil {
		return err
	}

	result, err := utils.QueryRow[VrfModel](c, InsertVrf, map[string]any{
		"sk":       payload.Sk,
		"pk":       payload.Pk,
		"pk_x":     payload.PkX,
		"pk_y":     payload.PkY,
		"key_hash": payload.KeyHash,
		"chain_id": chain_result.ChainId})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	payload := new(struct {
		CHAIN string `db:"name" json:"chain"`
	})

	if len(c.Body()) == 0 {
		results, err := utils.QueryRows[VrfModel](c, GetVrfWithoutChainId, nil)
		if err != nil {
			return err
		}
		return c.JSON(results)
	}

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": payload.CHAIN})
	if err != nil {
		return err
	}

	results, err := utils.QueryRows[VrfModel](c, GetVrf, map[string]any{"chain_id": chain_result.ChainId})
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[VrfModel](c, GetVrfById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(VrfUpdateModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[VrfModel](c, UpdateVrfById, map[string]any{
		"id":       id,
		"sk":       payload.Sk,
		"pk":       payload.Pk,
		"pk_x":     payload.PkX,
		"pk_y":     payload.PkY,
		"key_hash": payload.KeyHash})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[VrfModel](c, DeleteVrfById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
