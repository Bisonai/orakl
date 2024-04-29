package fetcher

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	fetcher := router.Group("/fetcher")

	fetcher.Post("/start", start)
	fetcher.Post("/stop", stop)
	fetcher.Post("/refresh", refresh)
	fetcher.Post("/activate/:id", activate)
	fetcher.Post("/deactivate/:id", deactivate)
}
