package apierr

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	apierr := router.Group("/error")

	apierr.Post("", insert)
	apierr.Get("", get)
	apierr.Get("/:id", getById)
	apierr.Delete("/:id", deleteById)
}
