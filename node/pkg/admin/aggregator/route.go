package aggregator

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	aggregator := router.Group("/aggregator")

	aggregator.Post("", insert)
	aggregator.Get("", get)

	aggregator.Post("/start", start)
	aggregator.Post("/stop", stop)
	aggregator.Post("/refresh", refresh)

	aggregator.Post("/sync/adapter", syncWithAdapter)
	aggregator.Post("/sync/config", SyncFromOraklConfig)
	aggregator.Post("/sync/config/:name", addFromOraklConfig)
	aggregator.Get("/:id", getById)
	aggregator.Delete("/:id", deleteById)
	aggregator.Post("/activate/:id", activate)
	aggregator.Post("/deactivate/:id", deactivate)

}
