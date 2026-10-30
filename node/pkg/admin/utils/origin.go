package utils

import (
	"github.com/gofiber/fiber/v2"
)

// RejectCrossOrigin answers 403 to state-changing requests (every method except
// GET, HEAD and OPTIONS) that carry an Origin header or a Sec-Fetch-Site other than "none". Browsers
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
		// Sec-Fetch-Site "none" is a user-initiated navigation, not a cross-site request.
		if site := c.Get("Sec-Fetch-Site"); c.Get(fiber.HeaderOrigin) != "" || (site != "" && site != "none") {
			return fiber.NewError(fiber.StatusForbidden, "browser requests are not allowed")
		}
		return c.Next()
	}
}
