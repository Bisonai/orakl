package api

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	api := router.Group("")

	api.Get("/symbols", getSymbols)
	api.Get("/latest-data-feeds/all", getAllLatestFeeds)
	api.Get("/latest-data-feeds/bulk/all", getAllLatestFeedsBulk)
	api.Get("/latest-data-feeds/bulk/:symbols", getLatestFeedsBulk)
	api.Get("/latest-data-feeds/:symbols", getLatestFeedsBulk)
	api.Get("/ws", websocket.New(HandleWebsocket))
}
