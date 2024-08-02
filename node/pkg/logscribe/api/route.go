package api

import "github.com/gofiber/fiber/v2"

func Routes(router fiber.Router, logsChannel chan []LogInsertModel) {
	api := router.Group("")
	api.Post("/", func(c *fiber.Ctx) error {
		return insertLogs(c, logsChannel)
	})
}
