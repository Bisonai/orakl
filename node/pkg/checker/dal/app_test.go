package dal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchConfigsRejectsEmptyBundle(t *testing.T) {
	for _, body := range []string{`[]`, `null`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(srv.Close)
			t.Setenv("CONFIG_BASE_URL", srv.URL)
			t.Setenv("CONFIG_FALLBACK_URL", "")
			t.Setenv("CHAIN", "mainnet")

			configs, err := fetchConfigs()
			require.ErrorContains(t, err, "config bundle is empty")
			assert.Nil(t, configs)
		})
	}
}

func TestHandleWsMessage(t *testing.T) {
	// Create a mock context
	ctx := context.Background()

	// Define test cases
	tests := []struct {
		name        string
		inputData   map[string]interface{}
		expected    WsResponse
		expectError bool
	}{
		{
			name: "Valid data",
			inputData: map[string]interface{}{
				"symbol":        "BTC",
				"aggregateTime": "1625151600000",
			},
			expected: WsResponse{
				Symbol:        "BTC",
				AggregateTime: "1625151600000",
			},
			expectError: false,
		},
		{
			name: "Invalid data",
			inputData: map[string]interface{}{
				"symbol":        "BTC",
				"aggregateTime": 1625151600000, // Invalid type
			},
			expected:    WsResponse{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			err := handleWsMessage(ctx, tt.inputData)

			// Check for errors
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
