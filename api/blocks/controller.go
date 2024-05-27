package blocks

import (
	"bisonai.com/orakl/api/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ObservedBlockModel struct {
	Service     string `db:"service" json:"service" validate:"required"`
	BlockNumber int64  `db:"block_number" json:"blockNumber" validate:"isZeroOrPositive"`
}

func upsertObservedBlock(c *fiber.Ctx) error {
	payload := new(ObservedBlockModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	validate.RegisterValidation("isZeroOrPositive", func(fl validator.FieldLevel) bool {
		return fl.Field().Int() >= 0
	})
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ObservedBlockModel](c, UpsertObservedBlock, map[string]any{
		"service":      payload.Service,
		"block_number": payload.BlockNumber,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}