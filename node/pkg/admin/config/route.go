package config

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	config := router.Group("/config")
	config.Post("/sync", Sync)
	config.Get("", Get)
	config.Get("/:id", GetById)
	config.Delete("/:id", DeleteById)
}
