package service

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	service := router.Group("/service")

	service.Post("", insert)
	service.Get("", get)
	service.Get("/:id", getById)
	service.Patch("/:id", updateById)
	service.Delete("/:id", deleteById)
}
