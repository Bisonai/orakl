package vrf

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	vrf := router.Group("/vrf")

	vrf.Post("", insert)
	vrf.Get("", get)
	vrf.Get("/:id", getById)
	vrf.Patch("/:id", updateById)
	vrf.Delete("/:id", deleteById)
}
