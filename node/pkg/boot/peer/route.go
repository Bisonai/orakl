package peer

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	peer := router.Group("/peer")

	peer.Post("/sync", sync)
	peer.Post("", insert)
	peer.Get("", get)
}
