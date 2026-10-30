package utils

import (
	"github.com/gofiber/fiber/v2"
)

// RejectCrossOrigin answers 403 to state-changing requests (every method except
// GET, HEAD and OPTIONS) that carry an Origin or Sec-Fetch-Site header. Browsers
// send these on such requests, including `no-cors` simple POSTs that skip
// preflight, while curl, Go clients and `nodebin admin` do not. No browser UI
// uses the admin API, so this blocks writes from web pages whether or not
// ADMIN_TOKEN is set.
func RejectCrossOrigin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		if c.Get(fiber.HeaderOrigin) != "" || c.Get("Sec-Fetch-Site") != "" {
			return fiber.NewError(fiber.StatusForbidden, "browser requests are not allowed")
		}
		return c.Next()
	}
}
