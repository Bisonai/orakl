package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const testToken = "s3cret-token"

// Sample admin routes for middleware behaviour; pkg/admin tests enumerate the real ones.
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
	// OPTIONS is safe; unregistered here so fiber answers 405, but never 401.
	if got := do(t, app, http.MethodOptions, protectedRoutes[0], ""); got == http.StatusUnauthorized {
		t.Errorf("OPTIONS %s: got 401", protectedRoutes[0])
	}
}

// A token with surrounding whitespace (e.g. secret file with trailing newline) still matches.
func TestAdminAuthTokenWhitespace(t *testing.T) {
	app := newTestApp(" " + testToken + "\n")
	if got := do(t, app, http.MethodPost, protectedRoutes[0], "Bearer "+testToken); got != http.StatusOK {
		t.Errorf("got %d, want 200", got)
	}
	if got := do(t, app, http.MethodPost, protectedRoutes[0], ""); got != http.StatusUnauthorized {
		t.Errorf("no header: got %d, want 401", got)
	}
}

// Whitespace-only token disables auth like an unset one.
func TestAdminAuthBlankToken(t *testing.T) {
	app := newTestApp("  \n")
	if got := do(t, app, http.MethodPost, protectedRoutes[0], "Bearer  "); got != http.StatusOK {
		t.Errorf("got %d, want 200", got)
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

func TestResolveAdminToken(t *testing.T) {
	cases := []struct {
		token, required string
		want            string
		wantErr         bool
	}{
		{"", "", "", false},
		{"", "false", "", false},
		{"", "0", "", false},
		{testToken, "", testToken, false},
		{" " + testToken + "\n", "true", testToken, false},
		{testToken, "1", testToken, false},
		{testToken, " true\n", testToken, false},
		{"", "true", "", true},
		{"", "1", "", true},
		{" \n", "true", "", true},
		// strict: anything else is a misconfiguration, even with a token
		{testToken, "TRUE", "", true},
		{testToken, "yes", "", true},
		{"", "on", "", true},
	}
	for _, tc := range cases {
		got, err := resolveAdminToken(tc.token, tc.required)
		if (err != nil) != tc.wantErr {
			t.Errorf("resolveAdminToken(%q, %q): err %v, wantErr %v", tc.token, tc.required, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("resolveAdminToken(%q, %q) = %q, want %q", tc.token, tc.required, got, tc.want)
		}
	}
	if _, err := resolveAdminToken("", "true"); !errors.Is(err, ErrAdminTokenRequired) {
		t.Errorf("required + empty token: got %v, want ErrAdminTokenRequired", err)
	}
}

// AdminToken reads both env vars (secrets falls back to env when Vault is not initialized).
func TestAdminTokenFromEnv(t *testing.T) {
	t.Setenv(AdminTokenEnv, "")
	t.Setenv(AdminAuthRequiredEnv, "true")
	if _, err := AdminToken(); !errors.Is(err, ErrAdminTokenRequired) {
		t.Errorf("got %v, want ErrAdminTokenRequired", err)
	}

	t.Setenv(AdminTokenEnv, testToken)
	if got, err := AdminToken(); err != nil || got != testToken {
		t.Errorf("got %q, %v; want %q, nil", got, err, testToken)
	}

	t.Setenv(AdminTokenEnv, "")
	t.Setenv(AdminAuthRequiredEnv, "")
	if got, err := AdminToken(); err != nil || got != "" {
		t.Errorf("unset: got %q, %v; want \"\", nil", got, err)
	}
}
