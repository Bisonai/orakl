package adapter

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	adapter := router.Group("/adapter")

	adapter.Post("", insert)
	adapter.Post("/hash", hash)
	adapter.Get("", get)
	adapter.Get("/:id", getById)
	adapter.Delete("/:id", deleteById)
}
