package utils

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"strings"

	"bisonai.com/miko/node/pkg/secrets"
	"github.com/gofiber/fiber/v2"
)

// AdminTokenEnv names the secret (Vault or env) holding the admin API bearer token.
const AdminTokenEnv = "ADMIN_TOKEN"

// AdminAuthRequiredEnv names the env var that makes ADMIN_TOKEN mandatory.
// Accepted values: "true"/"1" (required), "false"/"0" or unset (optional).
const AdminAuthRequiredEnv = "ADMIN_AUTH_REQUIRED"

// ErrAdminTokenRequired is returned when ADMIN_AUTH_REQUIRED is set but the
// resolved ADMIN_TOKEN is empty (unset, blank, or Vault read failed).
var ErrAdminTokenRequired = errors.New(AdminAuthRequiredEnv + " is true but " + AdminTokenEnv + " is empty or unreadable")

// AdminToken returns the trimmed ADMIN_TOKEN (Vault or env). It fails if
// ADMIN_AUTH_REQUIRED is invalid, or true while the token is empty, so callers
// never run (or call) the admin API unauthenticated by accident.
func AdminToken() (string, error) {
	return resolveAdminToken(secrets.GetSecret(AdminTokenEnv), os.Getenv(AdminAuthRequiredEnv))
}

func resolveAdminToken(token, required string) (string, error) {
	token = strings.TrimSpace(token)
	var req bool
	switch strings.TrimSpace(required) {
	case "", "false", "0":
	case "true", "1":
		req = true
	default:
		return "", fmt.Errorf("invalid %s %q: want true, 1, false or 0", AdminAuthRequiredEnv, required)
	}
	if req && token == "" {
		return "", ErrAdminTokenRequired
	}
	return token, nil
}

// AdminAuth requires `Authorization: Bearer <token>` for state-changing requests
// (every method except GET, HEAD and OPTIONS). Read-only requests stay open so
// probes, metrics scraping and sentinel keep working. Surrounding whitespace in
// token is ignored. An empty token disables the check.
func AdminAuth(token string) fiber.Handler {
	token = strings.TrimSpace(token)
	if token == "" {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}
	want := sha256.Sum256([]byte(token))

	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
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
