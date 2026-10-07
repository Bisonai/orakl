//nolint:all
package health

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectHealthCheckJSON(t *testing.T) {
	// cypress and its post-flip alias mainnet must both pick the cypress bundle;
	// baobab and its post-flip alias kairos must both pick the baobab bundle.
	cypress := []string{"cypress", "mainnet"}
	for _, chain := range cypress {
		raw, ok := selectHealthCheckJSON(chain)
		assert.True(t, ok, "chain %q should select a bundle", chain)
		assert.True(t, bytes.Equal(raw, cypressJSON), "chain %q should pick cypress_healthcheck.json", chain)
	}

	baobab := []string{"baobab", "kairos"}
	for _, chain := range baobab {
		raw, ok := selectHealthCheckJSON(chain)
		assert.True(t, ok, "chain %q should select a bundle", chain)
		assert.True(t, bytes.Equal(raw, baobabJSON), "chain %q should pick baobab_healthcheck.json", chain)
	}

	_, ok := selectHealthCheckJSON("invalid")
	assert.False(t, ok, "unknown chain should not select a bundle")
}

func TestCheckUrl(t *testing.T) {
	ctx := context.Background()
	// Test case 1: URL with "http" prefix
	httpUrl := HealthCheckUrl{Url: "http://example.com"}
	if !checkUrl(ctx, httpUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", httpUrl.Url)
	}

	// Test case 2: URL with "redis" prefix
	redisUrl := HealthCheckUrl{Url: "redis://localhost:6379"}
	if !checkUrl(ctx, redisUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", redisUrl.Url)
	}

	// Test case 3: Invalid URL
	invalidUrl := HealthCheckUrl{Url: "invalid-url"}
	if checkUrl(ctx, invalidUrl) {
		t.Errorf("checkUrl(%s) = true, expected false", invalidUrl.Url)
	}
}

func TestCheckHttp(t *testing.T) {
	// Test case 1: Valid URL with HTTP 200 response
	validUrl := "http://example.com"
	if !checkHttp(validUrl) {
		t.Errorf("checkHttp(%s) = false, expected true", validUrl)
	}

	// Test case 2: Valid URL with non-200 response
	non200Url := "http://example.com/nonexistent"
	if checkHttp(non200Url) {
		t.Errorf("checkHttp(%s) = true, expected false", non200Url)
	}

	// Test case 3: Invalid URL
	invalidUrl := "invalid-url"
	if checkHttp(invalidUrl) {
		t.Errorf("checkHttp(%s) = true, expected false", invalidUrl)
	}
}

func TestCheckRedis(t *testing.T) {
	ctx := context.Background()
	url := "redis://localhost:6379"

	result := checkRedis(ctx, url)

	assert.True(t, result)
}
