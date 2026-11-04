package metrics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Routes serves Prometheus metrics, incl. per-config aggregate gauges
// (pkg/metrics) read by the sentinel offset checker.
func Routes(router fiber.Router) {
	router.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}
