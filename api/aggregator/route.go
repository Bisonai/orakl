package aggregator

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	aggregator := router.Group("/aggregator")

	aggregator.Post("", insert)
	aggregator.Post("/hash", hash)
	aggregator.Get("", get)
	aggregator.Get("/:hash/:chain", getByHashAndChain)
	aggregator.Delete("/:id", deleteById)
	aggregator.Patch("/:hash", updateByHash)
}
