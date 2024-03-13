package reporter

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	reporter := router.Group("/reporter")

	reporter.Post("/activate", activate)
	reporter.Post("/deactivate", deactivate)
	reporter.Post("/refresh", refresh)
}
