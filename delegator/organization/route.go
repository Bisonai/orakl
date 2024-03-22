package organization

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	organization := router.Group("/organization")

	organization.Post("", insert)
	organization.Get("", get)
	organization.Get("/:id", getById)
	organization.Patch("/:id", updateById)
	organization.Delete("/:id", deleteById)
}
