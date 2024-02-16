package aggregate

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	aggregate := router.Group("/aggregate")

	aggregate.Post("", insert)
	aggregate.Get("", get)
	aggregate.Get("/:id", getById)
	aggregate.Get("/hash/:hash/latest", getLatestByHash)
	aggregate.Get("/id/:id/latest", getLatestById)
	aggregate.Patch("/:id", updateById)
	aggregate.Delete("/:id", deleteById)
}
