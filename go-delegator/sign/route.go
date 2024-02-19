package sign

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	sign := router.Group("/sign")

	sign.Post("", insert)
	sign.Get("/initialize", initialize)
	sign.Get("", get)
	sign.Get("/:id", getById)
}
