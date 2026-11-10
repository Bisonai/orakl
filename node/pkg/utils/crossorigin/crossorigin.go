// Package crossorigin holds the shared browser-request guard for the HTTP
// services that have no browser UI (admin, api, boot-api, delegator).
package crossorigin

import (
	"github.com/gofiber/fiber/v2"
)

// Reject answers 403 to state-changing requests (every method except GET, HEAD
// and OPTIONS) that carry an Origin header or a Sec-Fetch-Site other than
// "none". Browsers send these on such requests, including `no-cors` simple
// POSTs that skip preflight, while curl, Go and Node clients do not. Dropping
// CORS alone does not stop those simple POSTs, so services without a browser
// consumer use this instead of a CORS middleware.
func Reject() fiber.Handler {
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
