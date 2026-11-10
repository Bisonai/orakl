package crossorigin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newApp() *fiber.App {
	app := fiber.New()
	app.Use(Reject())
	ok := func(c *fiber.Ctx) error { return c.SendString("ok") }
	app.All("/x", ok)
	return app
}

func status(t *testing.T, app *fiber.App, method string, headers map[string]string) int {
	t.Helper()
	req := httptest.NewRequest(method, "/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

var writes = []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

func TestRejectBrowserWrites(t *testing.T) {
	app := newApp()
	for _, m := range writes {
		for _, origin := range []string{"http://evil.example", "null", "http://localhost:8088"} {
			if got := status(t, app, m, map[string]string{fiber.HeaderOrigin: origin, fiber.HeaderContentType: "text/plain"}); got != http.StatusForbidden {
				t.Errorf("%s origin %q: got %d, want 403", m, origin, got)
			}
		}
		for _, site := range []string{"cross-site", "same-site", "same-origin"} {
			if got := status(t, app, m, map[string]string{"Sec-Fetch-Site": site}); got != http.StatusForbidden {
				t.Errorf("%s Sec-Fetch-Site %q: got %d, want 403", m, site, got)
			}
		}
	}
}

func TestAllowNonBrowserWrites(t *testing.T) {
	app := newApp()
	for _, m := range writes {
		if got := status(t, app, m, nil); got != http.StatusOK {
			t.Errorf("%s without browser headers: got %d, want 200", m, got)
		}
		if got := status(t, app, m, map[string]string{"Sec-Fetch-Site": "none"}); got != http.StatusOK {
			t.Errorf("%s Sec-Fetch-Site none: got %d, want 200", m, got)
		}
	}
}

func TestAllowReads(t *testing.T) {
	app := newApp()
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if got := status(t, app, m, map[string]string{fiber.HeaderOrigin: "http://evil.example"}); got != http.StatusOK {
			t.Errorf("%s with Origin: got %d, want 200", m, got)
		}
	}
}
