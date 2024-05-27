package blocks

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	blocks := router.Group("/blocks")

	blocks.Post("/observed", upsertObservedBlock)
	// blocks.Post("/unprocessed", upsertUnprocessedBlock)
	// blocks.Get("/unprocessed", getUnprocessedBlocks)
	// blocks.Delete("/unprocessed", deleteUnprocessedBlock)
}