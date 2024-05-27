package blocks

import (
	"bisonai.com/orakl/api/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type BlockModel struct {
	Service     string `db:"service" json:"service" validate:"required"`
	BlockNumber int64  `db:"block_number" json:"blockNumber" validate:"isZeroOrPositive"`
}

func upsertObservedBlock(c *fiber.Ctx) error {
	payload := new(BlockModel)
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

	result, err := utils.QueryRow[BlockModel](c, UpsertObservedBlock, map[string]any{
		"service":      payload.Service,
		"block_number": payload.BlockNumber,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func getUnprocessedBlocks(c *fiber.Ctx) error {
	service := c.Query("service")
	if service == "" {
		return fiber.NewError(fiber.StatusBadRequest, "service is required")
	}
	result, err := utils.QueryRows[BlockModel](c, GetUnprocessedBlocks, map[string]any{
		"service": service,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func insertUnprocessedBlock(c *fiber.Ctx) error {
	payload := new(BlockModel)
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

	result, err := utils.QueryRow[BlockModel](c, InsertUnprocessedBlock, map[string]any{
		"service":      payload.Service,
		"block_number": payload.BlockNumber,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteUnprocessedBlock(c *fiber.Ctx) error {
	service := c.Params("service")
	blockNumber := c.Params("blockNumber")
	
	result, err := utils.QueryRow[BlockModel](c, DeleteUnprocessedBlock, map[string]any{
		"service": service,
		"block_number": blockNumber,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}