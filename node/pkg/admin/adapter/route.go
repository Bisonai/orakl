package adapter

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	adapter := router.Group("/adapter")

	adapter.Post("", insert)
	adapter.Get("", get)
	adapter.Post("/sync", syncFromOraklConfig)
	adapter.Post("/sync/:name", addFromOraklConfig)
	adapter.Get("/detail/:id", getDetailById)
	adapter.Get("/:id", getById)
	adapter.Delete("/:id", deleteById)
	adapter.Post("/activate/:id", activate)
	adapter.Post("/deactivate/:id", deactivate)
}
