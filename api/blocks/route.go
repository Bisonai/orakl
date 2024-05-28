package blocks

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	blocks := router.Group("/blocks")

	blocks.Get("/observed", getObservedBlock)
	blocks.Post("/observed", upsertObservedBlock)
	blocks.Post("/unprocessed", insertUnprocessedBlocks)
	blocks.Get("/unprocessed", getUnprocessedBlocks)
	blocks.Delete("/unprocessed/:service/:blockNumber", deleteUnprocessedBlock)
}