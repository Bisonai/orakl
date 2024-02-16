package data

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	data := router.Group("/data")

	data.Post("", bulkInsert)
	data.Get("", get)
	data.Get("/:id", getById)
	data.Delete("/:id", deleteById)
	data.Get("/feed/:id", getByFeedId)
}
