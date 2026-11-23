package main

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/utils/buildinfo"
	"github.com/gofiber/fiber/v2"
)

func TestRootAndVersion(t *testing.T) {
	app := fiber.New()
	SetSigningRouter(app.Group("/api/v1"))

	get := func(path string) (int, string) {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil), -1)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}

	// health checks match the root text by substring: keep the exact prefix
	status, body := get("/api/v1")
	if status != fiber.StatusOK || !strings.HasPrefix(body, "Orakl Network Delegator ") || body != buildinfo.Root("Orakl Network Delegator") {
		t.Fatalf("root: %d %q", status, body)
	}

	status, body = get("/api/v1/version")
	var info buildinfo.Info
	if status != fiber.StatusOK || json.Unmarshal([]byte(body), &info) != nil || info != buildinfo.Get() {
		t.Fatalf("version: %d %q", status, body)
	}
}
