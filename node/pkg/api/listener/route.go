package listener

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	listener := router.Group("/listener")

	listener.Post("", insert)
	listener.Get("", get)
	listener.Get("/:id", getById)
	listener.Patch("/:id", updateById)
	listener.Delete("/:id", deleteById)
}
