package aggregator

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	aggregator := router.Group("/aggregator")

	aggregator.Post("/start", start)
	aggregator.Post("/stop", stop)
	aggregator.Post("/refresh", refresh)
	aggregator.Post("/activate/:id", activate)
	aggregator.Post("/deactivate/:id", deactivate)
	aggregator.Post("/renew-signer", renewSigner)
	aggregator.Get("/signer", getSigner)
}
