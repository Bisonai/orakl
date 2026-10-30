package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newOriginTestApp(token string) *fiber.App {
	app := fiber.New()
	app.Use(RejectCrossOrigin())
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

func doOrigin(t *testing.T, app *fiber.App, method, path, origin, auth, contentType string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if origin != "" {
		req.Header.Set(fiber.HeaderOrigin, origin)
	}
	if auth != "" {
		req.Header.Set(fiber.HeaderAuthorization, auth)
	}
	if contentType != "" {
		req.Header.Set(fiber.HeaderContentType, contentType)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

// A no-cors simple POST from a web page carries Origin and is rejected, with or without a token.
func TestRejectCrossOriginStateChanging(t *testing.T) {
	for _, token := range []string{"", testToken} {
		app := newOriginTestApp(token)
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			for _, p := range protectedRoutes {
				for _, origin := range []string{"http://evil.example", "null", "http://localhost:8088"} {
					for _, auth := range []string{"", "Bearer " + testToken} {
						if got := doOrigin(t, app, method, p, origin, auth, "text/plain"); got != http.StatusForbidden {
							t.Errorf("token %q: %s %s (origin %q, auth %q): got %d, want 403", token, method, p, origin, auth, got)
						}
					}
				}
			}
		}
	}
}

// A browser request that omits Origin but carries Sec-Fetch-Site is rejected too.
func TestRejectCrossOriginSecFetchSite(t *testing.T) {
	app := newOriginTestApp("")
	for _, site := range []string{"cross-site", "same-site", "same-origin"} {
		req := httptest.NewRequest(http.MethodPost, protectedRoutes[0], nil)
		req.Header.Set("Sec-Fetch-Site", site)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Sec-Fetch-Site %q: %v", site, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Sec-Fetch-Site %q: got %d, want 403", site, resp.StatusCode)
		}
	}
}

// Sec-Fetch-Site "none" without Origin is not cross-site.
func TestRejectCrossOriginSecFetchSiteNone(t *testing.T) {
	app := newOriginTestApp("")
	req := httptest.NewRequest(http.MethodPost, protectedRoutes[0], nil)
	req.Header.Set("Sec-Fetch-Site", "none")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got %d, want 200", resp.StatusCode)
	}
}

// Non-browser clients (no Origin) are unaffected.
func TestRejectCrossOriginNoOrigin(t *testing.T) {
	app := newOriginTestApp("")
	for _, p := range protectedRoutes {
		if got := doOrigin(t, app, http.MethodPost, p, "", "", ""); got != http.StatusOK {
			t.Errorf("no token: POST %s: got %d, want 200", p, got)
		}
	}

	app = newOriginTestApp(testToken)
	for _, p := range protectedRoutes {
		if got := doOrigin(t, app, http.MethodPost, p, "", "Bearer "+testToken, ""); got != http.StatusOK {
			t.Errorf("valid token: POST %s: got %d, want 200", p, got)
		}
		if got := doOrigin(t, app, http.MethodPost, p, "", "", ""); got != http.StatusUnauthorized {
			t.Errorf("missing token: POST %s: got %d, want 401", p, got)
		}
	}
}

// Read-only requests stay open even from browsers (probes, metrics, sentinel).
func TestRejectCrossOriginReadOnly(t *testing.T) {
	app := newOriginTestApp(testToken)
	for _, p := range openRoutes {
		for _, origin := range []string{"", "http://evil.example"} {
			if got := doOrigin(t, app, http.MethodGet, p, origin, "", ""); got != http.StatusOK {
				t.Errorf("GET %s (origin %q): got %d, want 200", p, origin, got)
			}
			if got := doOrigin(t, app, http.MethodHead, p, origin, "", ""); got != http.StatusOK {
				t.Errorf("HEAD %s (origin %q): got %d, want 200", p, origin, got)
			}
		}
	}
}
