package feed

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	feed := router.Group("/feed")
	feed.Get("", get)
	feed.Get("/adapter/:id", getByAdpaterId)
	feed.Get("/:id", getById)

}
