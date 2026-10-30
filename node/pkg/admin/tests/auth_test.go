//nolint:all
package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// Setup wires ADMIN_TOKEN into the admin app.
func TestSetupAdminToken(t *testing.T) {
	t.Setenv(utils.AdminTokenEnv, "s3cret-token")

	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	status := func(method, path, auth string) int {
		req := httptest.NewRequest(method, path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := testItems.app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	assert.Equal(t, http.StatusUnauthorized, status(http.MethodPost, "/api/v1/host/sync", ""))
	assert.Equal(t, http.StatusUnauthorized, status(http.MethodPost, "/api/v1/host/sync", "Bearer wrong"))

	channel := testItems.mb.Subscribe(bus.LIBP2P)
	waitForMessage(t, channel, bus.ADMIN, bus.LIBP2P, bus.LIBP2P_SYNC)
	assert.Equal(t, http.StatusOK, status(http.MethodPost, "/api/v1/host/sync", "Bearer s3cret-token"))

	waitForMessageWithResponse(t, channel, bus.ADMIN, bus.LIBP2P, bus.GET_PEER_COUNT, map[string]any{"Count": 1})
	assert.Equal(t, http.StatusOK, status(http.MethodGet, "/api/v1/host/peercount", ""))

	// No CORS middleware (#2725): preflight gets no Access-Control-Allow-* headers.
	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/host/sync", nil)
	preflight.Header.Set("Origin", "http://example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	resp, err := testItems.app.Test(preflight, -1)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))

	// Browser POST (Origin set) is rejected even with a valid token.
	assert.Equal(t, http.StatusForbidden, statusWithOrigin(t, testItems.app, http.MethodPost, "/api/v1/host/sync", "http://example.com", "Bearer s3cret-token"))
}

// Setup rejects browser (Origin) state-changing requests when ADMIN_TOKEN is unset;
// non-browser POSTs and GETs keep working.
func TestSetupRejectCrossOrigin(t *testing.T) {
	t.Setenv(utils.AdminTokenEnv, "")

	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	// no-cors simple POST from a web page: text/plain, Origin set, no preflight.
	assert.Equal(t, http.StatusForbidden, statusWithOrigin(t, testItems.app, http.MethodPost, "/api/v1/aggregator/stop", "http://evil.example", ""))
	assert.Equal(t, http.StatusForbidden, statusWithOrigin(t, testItems.app, http.MethodPost, "/api/v1/host/sync", "null", ""))

	channel := testItems.mb.Subscribe(bus.LIBP2P)
	waitForMessage(t, channel, bus.ADMIN, bus.LIBP2P, bus.LIBP2P_SYNC)
	assert.Equal(t, http.StatusOK, statusWithOrigin(t, testItems.app, http.MethodPost, "/api/v1/host/sync", "", ""))

	assert.Equal(t, http.StatusOK, statusWithOrigin(t, testItems.app, http.MethodGet, "/api/v1/config", "", ""))
	assert.Equal(t, http.StatusOK, statusWithOrigin(t, testItems.app, http.MethodGet, "/api/v1/config", "http://evil.example", ""))
}

func statusWithOrigin(t *testing.T, app *fiber.App, method, path, origin, auth string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader("x"))
	req.Header.Set("Content-Type", "text/plain")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
