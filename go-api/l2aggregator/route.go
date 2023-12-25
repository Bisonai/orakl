package l2aggregator

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	l2aggregator := router.Group("/l2aggregator")

	l2aggregator.Get("/:chain/:l1Address", get)
}
