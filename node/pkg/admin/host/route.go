package host

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	host := router.Group("/host")

	host.Get("/peercount", getPeerCount)
	host.Post("/sync", sync)
}
