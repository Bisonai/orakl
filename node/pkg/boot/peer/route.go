package peer

import (
	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router, store *peerstore.Store) {
	peer := router.Group("/peer")

	peer.Post("/sync", syncHandler(store))
}
