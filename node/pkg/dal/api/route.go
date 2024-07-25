package api

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	api := router.Group("")

	api.Get("/symbols", getSymbols)
	api.Get("/latest-data-feeds/all", getAllLatestFeeds)
	api.Get("/latest-data-feeds/transpose/all", getAllLatestFeedsTransposed)
	api.Get("/latest-data-feeds/transpose/:symbols", getLatestFeedsTransposed)
	api.Get("/latest-data-feeds/:symbols", getLatestFeeds)
	api.Get("/ws", websocket.New(HandleWebsocket))
}
