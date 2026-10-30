//nolint:all
package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
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

	// CORS preflight is answered by cors (registered before auth) without a token.
	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/host/sync", nil)
	preflight.Header.Set("Origin", "http://example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	resp, err := testItems.app.Test(preflight, -1)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
