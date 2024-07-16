package api

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	api := router.Group("/dal")

	api.Get("/symbols", getSymbols)
	api.Get("/latest-data-feeds/all", getLatestFeeds)
	api.Get("/latest-data-feeds/:symbols", getLatestMultipleFeeds)
	api.Get("/ws", websocket.New(HandleWebsocket))
}
