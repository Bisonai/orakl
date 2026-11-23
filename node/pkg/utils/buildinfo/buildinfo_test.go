package buildinfo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDefaults(t *testing.T) {
	if Version != "dev" || Commit != "unknown" || BuildTime != "unknown" {
		t.Fatalf("unexpected defaults: %q %q %q", Version, Commit, BuildTime)
	}
	if got, want := Root("Miko Node DAL API"), "Miko Node DAL API dev (unknown)"; got != want {
		t.Fatalf("Root = %q, want %q", got, want)
	}
}

func TestRegisterMetricAndHandlers(t *testing.T) {
	Register("dal")
	Register("other") // ignored: first call wins

	want := `
# HELP orakl_build_info Build information of the running binary; always 1.
# TYPE orakl_build_info gauge
orakl_build_info{binary="dal",commit="unknown",version="dev"} 1
`
	if err := testutil.GatherAndCompare(prometheus.DefaultGatherer, strings.NewReader(want), "orakl_build_info"); err != nil {
		t.Fatal(err)
	}

	wantInfo := Info{Binary: "dal", Version: "dev", Commit: "unknown", Built: "unknown"}

	rec := httptest.NewRecorder()
	HTTPHandler(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	var got Info
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != wantInfo {
		t.Fatalf("HTTPHandler = %+v, want %+v", got, wantInfo)
	}

	app := fiber.New()
	app.Get("/version", FiberHandler)
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/version", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	got = Info{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got != wantInfo {
		t.Fatalf("FiberHandler = %+v, want %+v", got, wantInfo)
	}
}
