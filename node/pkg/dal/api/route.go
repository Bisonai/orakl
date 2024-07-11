package api

import (
	"context"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	api := router.Group("/dal")

	api.Get("/symbols", getSymbols)
	api.Get("/latest-data-feeds/all", getLatestFeeds)
	api.Get("/latest-data-feeds/:symbol", getLatestFeed)
	api.Get("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return websocket.New(func(conn *websocket.Conn) {
				ApiController.handleWebsocket(context.Background(), conn)
			})(c)
		}
		return fiber.ErrUpgradeRequired
	})
}
