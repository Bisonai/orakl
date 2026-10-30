package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const testToken = "s3cret-token"

// Mirrors the routes registered by admin.Run (/api/v1 group).
var protectedRoutes = []string{
	"/api/v1/fetcher/start",
	"/api/v1/fetcher/stop",
	"/api/v1/fetcher/refresh",
	"/api/v1/aggregator/start",
	"/api/v1/aggregator/stop",
	"/api/v1/aggregator/refresh",
	"/api/v1/aggregator/activate/1",
	"/api/v1/aggregator/deactivate/1",
	"/api/v1/aggregator/renew-signer",
	"/api/v1/config/sync",
	"/api/v1/host/sync",
}

var openRoutes = []string{
	"/api/v1/",
	"/api/v1/metrics",
	"/api/v1/host/peercount",
	"/api/v1/aggregator/signer",
	"/api/v1/config",
}

func newTestApp(token string) *fiber.App {
	app := fiber.New()
	app.Use(AdminAuth(token))
	ok := func(c *fiber.Ctx) error { return c.SendString("ok") }
	for _, p := range protectedRoutes {
		app.Post(p, ok)
		app.Put(p, ok)
		app.Patch(p, ok)
		app.Delete(p, ok)
	}
	for _, p := range openRoutes {
		app.Get(p, ok)
	}
	return app
}

func do(t *testing.T, app *fiber.App, method, path, auth string) int {
	t.Helper()
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

func TestAdminAuthProtectedRoutes(t *testing.T) {
	app := newTestApp(testToken)

	cases := []struct {
		name string
		auth string
		want int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"token prefix", "Bearer " + testToken[:4], http.StatusUnauthorized},
		{"token suffix", "Bearer " + testToken + "x", http.StatusUnauthorized},
		{"no scheme", testToken, http.StatusUnauthorized},
		{"basic scheme", "Basic " + testToken, http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"valid", "Bearer " + testToken, http.StatusOK},
		{"valid lowercase scheme", "bearer " + testToken, http.StatusOK},
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, p := range protectedRoutes {
			for _, tc := range cases {
				if got := do(t, app, method, p, tc.auth); got != tc.want {
					t.Errorf("%s %s (%s): got %d, want %d", method, p, tc.name, got, tc.want)
				}
			}
		}
	}
}

func TestAdminAuthOpenRoutes(t *testing.T) {
	app := newTestApp(testToken)
	for _, p := range openRoutes {
		for _, auth := range []string{"", "Bearer wrong", "Bearer " + testToken} {
			if got := do(t, app, http.MethodGet, p, auth); got != http.StatusOK {
				t.Errorf("GET %s (auth %q): got %d, want 200", p, auth, got)
			}
		}
		if got := do(t, app, http.MethodHead, p, ""); got != http.StatusOK {
			t.Errorf("HEAD %s: got %d, want 200", p, got)
		}
	}
}

func TestAdminAuthUnsetToken(t *testing.T) {
	app := newTestApp("")
	for _, p := range protectedRoutes {
		for _, auth := range []string{"", "Bearer anything"} {
			if got := do(t, app, http.MethodPost, p, auth); got != http.StatusOK {
				t.Errorf("POST %s (auth %q): got %d, want 200", p, auth, got)
			}
		}
	}
	for _, p := range openRoutes {
		if got := do(t, app, http.MethodGet, p, ""); got != http.StatusOK {
			t.Errorf("GET %s: got %d, want 200", p, got)
		}
	}
}
