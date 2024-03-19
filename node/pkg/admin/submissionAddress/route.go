package submissionAddress

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	submissionAddress := router.Group("/submission-address")

	submissionAddress.Post("/sync", SyncFromOraklConfig)
	submissionAddress.Post("", insert)
	submissionAddress.Get("", get)
	submissionAddress.Get("/:id", getById)
	submissionAddress.Delete("/:id", deleteById)
	submissionAddress.Patch("/:id", updateById)
}
