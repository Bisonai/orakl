package providerUrl

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	providerUrls := router.Group("/provider-url")

	providerUrls.Post("", insert)
	providerUrls.Get("", get)
	providerUrls.Get("/:id", getById)
	providerUrls.Delete("/:id", deleteById)
}
