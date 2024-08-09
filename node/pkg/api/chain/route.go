package chain

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	chain := router.Group("/chain")

	chain.Get("", get)
	chain.Get("/:id", getById)
	chain.Post("", insert)
	chain.Patch("/:id", patchById)
	chain.Delete("/:id", deleteById)
}
