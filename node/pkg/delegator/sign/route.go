package sign

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	sign := router.Group("/sign")

	sign.Post("/v2", insertV2)

	sign.Get("/initialize", initialize)
	sign.Get("/feePayer", getFeePayerAddress)
	sign.Get("", get)
	sign.Get("/:id", getById)
}
