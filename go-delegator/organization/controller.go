package organization

import (
	"bisonai.com/orakl/go-delegator/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type OrganizationInsertModel struct {
	Name string `json:"name" db:"name" validate:"required"`
}

type OrganizationModel struct {
	OrganizationId utils.CustomInt64 `json:"id" db:"organization_id"`
	Name           string            `json:"name" db:"name"`
}

func insert(c *fiber.Ctx) error {
	payload := new(OrganizationInsertModel)
	if err := c.BodyParser(payload); err != nil {
		panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		panic(err)
	}

	result, err := utils.QueryRow[OrganizationModel](c, InsertOrganization, map[string]any{"name": payload.Name})
	if err != nil {
		panic(err)
	}
	return c.JSON(result)

}

func get(c *fiber.Ctx) error {
	result, err := utils.QueryRows[OrganizationModel](c, GetOrganization, nil)
	if err != nil {
		panic(err)
	}
	return c.JSON(result)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[OrganizationModel](c, GetOrganizationById, map[string]any{"id": id})
	if err != nil {
		panic(err)
	}
	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(OrganizationInsertModel)
	if err := c.BodyParser(payload); err != nil {
		panic(err)
	}
	result, err := utils.QueryRow[OrganizationModel](c, UpdateOrganization, map[string]any{"name": payload.Name, "id": id})
	if err != nil {
		panic(err)
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[OrganizationModel](c, DeleteOrganization, map[string]any{"id": id})
	if err != nil {
		panic(err)
	}
	return c.JSON(result)
}
