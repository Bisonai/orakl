package sign

import (
	"github.com/gofiber/fiber/v2"
)

// SigningRoutes registers the delegator's routes: signing, key reload and fee
// payer address, all behind the bearer token check.
func SigningRoutes(router fiber.Router) {
	// registered before the group so it wraps the group-wide token check
	// (c.Next runs the following matching routes) and counts auth rejections
	router.Post("/sign/v2", observeSign)

	sign := router.Group("/sign", RequireToken)

	sign.Post("/v2", insertV2)
	sign.Get("/initialize", initialize)
	sign.Get("/feePayer", getFeePayerAddress)
}
