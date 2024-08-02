package api

import "github.com/gofiber/fiber/v2"

func Routes(router fiber.Router) {
	api := router.Group("")
	api.Post("/", insertLogs)
}
