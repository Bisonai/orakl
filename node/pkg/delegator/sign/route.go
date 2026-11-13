package sign

import (
	"github.com/gofiber/fiber/v2"
)

// SigningRoutes registers only the routes that work without Postgres
// (SQLite mode): signing, key reload and fee payer address.
func SigningRoutes(router fiber.Router) {
	sign := router.Group("/sign", RequireToken)

	sign.Post("/v2", insertV2)

	sign.Get("/initialize", initialize)
	sign.Get("/feePayer", getFeePayerAddress)
}

func Routes(router fiber.Router) {
	sign := router.Group("/sign")

	sign.Post("/v2", insertV2)

	sign.Get("/initialize", initialize)
	sign.Get("/feePayer", getFeePayerAddress)
	sign.Get("", get)
	sign.Get("/:id", getById)
}
