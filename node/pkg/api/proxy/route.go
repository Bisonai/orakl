package proxy

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	proxy := router.Group("/proxy")

	proxy.Post("", insert)
	proxy.Get("", get)
	proxy.Get("/:id", getById)
	proxy.Patch("/:id", updateById)
	proxy.Delete("/:id", deleteById)
}
