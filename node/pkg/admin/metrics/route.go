package metrics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// Routes serves Prometheus metrics, incl. per-config aggregate gauges
// (pkg/metrics) read by the sentinel offset checker.
func Routes(router fiber.Router) {
	router.Get("/metrics", adaptor.HTTPHandler(handler))
}
