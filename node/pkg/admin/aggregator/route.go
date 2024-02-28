package aggregator

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	aggregator := router.Group("/aggregator")

	aggregator.Post("", insert)
	aggregator.Get("", get)
	aggregator.Post("/sync", syncWithAdapter)
	aggregator.Get("/:id", getById)
	aggregator.Delete("/:id", deleteById)
	aggregator.Post("/activate/:id", activate)
	aggregator.Post("/deactivate/:id", deactivate)

}
