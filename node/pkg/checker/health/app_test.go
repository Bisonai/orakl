//nolint:all
package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckUrl(t *testing.T) {
	// Test case 1: URL with "http" prefix
	httpUrl := HealthCheckUrl{Url: "http://example.com"}
	if !checkUrl(httpUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", httpUrl.Url)
	}

	// Test case 2: URL with "redis" prefix
	redisUrl := HealthCheckUrl{Url: "redis://localhost:6379"}
	if !checkUrl(redisUrl) {
		t.Errorf("checkUrl(%s) = false, expected true", redisUrl.Url)
	}

	// Test case 3: Invalid URL
	invalidUrl := HealthCheckUrl{Url: "invalid-url"}
	if checkUrl(invalidUrl) {
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
