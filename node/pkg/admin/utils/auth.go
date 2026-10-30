package utils

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AdminTokenEnv names the secret (Vault or env) holding the admin API bearer token.
const AdminTokenEnv = "ADMIN_TOKEN"

// AdminAuth requires `Authorization: Bearer <token>` for state-changing requests
// (every method except GET and HEAD). Read-only requests stay open so probes,
// metrics scraping and sentinel keep working. An empty token disables the check.
func AdminAuth(token string) fiber.Handler {
	if token == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}
	want := sha256.Sum256([]byte(token))

	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead:
			return c.Next()
		}

		got, ok := bearerToken(c.Get(fiber.HeaderAuthorization))
		// Hash both sides so the comparison is constant-time regardless of length.
		gotHash := sha256.Sum256([]byte(got))
		if !ok || subtle.ConstantTimeCompare(gotHash[:], want[:]) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "missing or invalid admin token")
		}
		return c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}
