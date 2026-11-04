package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/admin/utils"
	"github.com/gofiber/fiber/v2"
)

// Read-only routes reachable without ADMIN_TOKEN. Adding a GET route to the
// admin API requires adding it here, i.e. deciding it is safe to leave open.
var openRoutes = map[string]bool{
	"/api/v1/":                  true, // liveness/readiness probes
	"/api/v1/metrics":           true, // Prometheus, sentinel offset checker
	"/api/v1/host/peercount":    true, // sentinel peers checker
	"/api/v1/aggregator/signer": true, // public signer address + status
	"/api/v1/config":            true, // config list, no secrets
}

// Every route registered by Run: GETs must be allowlisted and open, everything
// else must reject requests without a valid token.
func TestRegisteredRoutesAuth(t *testing.T) {
	const token = "s3cret-token"
	app := fiber.New()
	app.Use(utils.AdminAuth(token))
	registerRoutes(app)

	status := func(method, path, auth string) int {
		req := httptest.NewRequest(method, path, nil)
		if auth != "" {
			req.Header.Set(fiber.HeaderAuthorization, auth)
		}
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode
	}

	var protected int
	for _, r := range app.GetRoutes(true) {
		path := strings.ReplaceAll(r.Path, ":id", "1")
		switch r.Method {
		case fiber.MethodGet, fiber.MethodHead:
			if !openRoutes[r.Path] {
				t.Errorf("%s %s is open without token but not in openRoutes allowlist", r.Method, r.Path)
				continue
			}
			if got := status(r.Method, path, ""); got == http.StatusUnauthorized {
				t.Errorf("%s %s: got 401 without token, want open", r.Method, r.Path)
			}
		default:
			protected++
			for _, auth := range []string{"", "Bearer wrong"} {
				if got := status(r.Method, path, auth); got != http.StatusUnauthorized {
					t.Errorf("%s %s (auth %q): got %d, want 401", r.Method, r.Path, auth, got)
				}
			}
		}
	}
	if protected == 0 {
		t.Fatal("no protected routes registered")
	}
}
