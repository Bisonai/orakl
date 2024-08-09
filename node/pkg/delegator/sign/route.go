package sign

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	sign := router.Group("/sign")

	sign.Post("", insert)
	sign.Post("/v2", insertV2)
	sign.Post("/volatile", onlySign)

	sign.Get("/initialize", initialize)
	sign.Get("/feePayer", getFeePayerAddress)
	sign.Get("", get)
	sign.Get("/:id", getById)
}
