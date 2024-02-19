package contract

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	contract := router.Group("/contract")

	contract.Post("", insert)
	contract.Get("", get)
	contract.Get("/:id", getById)
	contract.Post("/connectReporter", connectReporter)
	contract.Post("/disconnectReporter", disconnectReporter)
	contract.Patch("/:id", updateById)
	contract.Delete("/:id", deleteById)
}
