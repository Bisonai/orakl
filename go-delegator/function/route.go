package function

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	contract := router.Group("/function")

	contract.Post("", insert)
	contract.Get("", get)
	contract.Get("/:id", getById)
	contract.Patch("/:id", updateById)
	contract.Delete("/:id", deleteById)
}
