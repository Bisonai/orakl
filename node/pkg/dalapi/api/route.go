package api

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	api := router.Group("/dal")

	api.Get("/latest-data-feeds/all", getLatestFeeds)
	api.Get("/latest-data-feeds/:symbol", getLatestFeed)
}
