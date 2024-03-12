package wallet

import (
	"bisonai.com/orakl/node/pkg/db"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type WalletModel struct {
	Id *int64 `db:"id" json:"id"`
	Pk string `db:"pk" json:"pk"`
}

type WalletInsertModel struct {
	Pk string `json:"pk" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(WalletInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	result, err := db.QueryRow[WalletModel](c.Context(), InsertWallet, map[string]any{
		"pk": payload.Pk})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute insert wallet query: " + err.Error())
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[WalletModel](c.Context(), GetWallets, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get wallet query: " + err.Error())
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[WalletModel](c.Context(), GetWalletById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get wallet by id query: " + err.Error())
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(WalletInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	result, err := db.QueryRow[WalletModel](c.Context(), UpdateWalletById, map[string]any{"pk": payload.Pk, "id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute update wallet by id query: " + err.Error())
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[WalletModel](c.Context(), DeleteWalletById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute delete wallet by id query: " + err.Error())
	}
	return c.JSON(result)
}
