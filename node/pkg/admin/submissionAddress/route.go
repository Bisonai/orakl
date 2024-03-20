package submissionAddress

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	submissionAddress := router.Group("/submission-address")

	submissionAddress.Post("/sync/aggregator", syncWithAggregator)
	submissionAddress.Post("/sync/config", SyncFromOraklConfig)
	submissionAddress.Post("/sync/config/:name", addFromOraklConfig)
	submissionAddress.Post("", insert)
	submissionAddress.Get("", get)
	submissionAddress.Get("/:id", getById)
	submissionAddress.Delete("/:id", deleteById)
	submissionAddress.Patch("/:id", updateById)
}
