// Package buildinfo holds the version, commit and build time stamped into a
// binary at build time, and exposes them as a metric, a startup log line, a
// root-text suffix and a GET /version payload.
//
// Set the values with the linker, e.g.
//
//	go build -ldflags "-X bisonai.com/miko/node/pkg/utils/buildinfo.Version=v0.0.29.20261120.0227.846179b \
//	  -X bisonai.com/miko/node/pkg/utils/buildinfo.Commit=846179b \
//	  -X bisonai.com/miko/node/pkg/utils/buildinfo.BuildTime=2026-11-20T02:27:00Z"
//
// Without them a binary reports "dev" / "unknown".
package buildinfo

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Set via -ldflags -X; see the package doc.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Info is the GET /version payload.
type Info struct {
	Binary  string `json:"binary"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

var (
	mu       sync.RWMutex
	binary   = "unknown"
	register sync.Once
)

// Register records the binary name, exports orakl_build_info{binary,version,commit} 1
// on the default Prometheus registry and logs the build info once. Call it
// early in main, after the logger is initialised. Later calls only update the
// binary name reported by Get.
func Register(name string) {
	mu.Lock()
	binary = name
	mu.Unlock()

	register.Do(func() {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "orakl_build_info",
			Help: "Build information of the running binary; always 1.",
		}, []string{"binary", "version", "commit"})
		g.WithLabelValues(name, Version, Commit).Set(1)
		if err := prometheus.Register(g); err != nil {
			log.Warn().Err(err).Msg("failed to register orakl_build_info")
		}
		log.Info().
			Str("binary", name).
			Str("version", Version).
			Str("commit", Commit).
			Str("built", BuildTime).
			Msg("build info")
	})
}

// Get returns the build info of the running binary.
func Get() Info {
	mu.RLock()
	defer mu.RUnlock()
	return Info{Binary: binary, Version: Version, Commit: Commit, Built: BuildTime}
}

// Root appends the version and commit to an HTTP root text, keeping text as
// an exact prefix (health checks match on it), e.g.
// "Miko Node DAL API v0.0.29.20261120.0227.846179b (846179b)".
func Root(text string) string {
	return text + " " + Version + " (" + Commit + ")"
}

// HTTPHandler serves Get as JSON.
func HTTPHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Get()); err != nil {
		log.Error().Err(err).Msg("failed to write version response")
	}
}

// FiberHandler serves Get as JSON.
func FiberHandler(c *fiber.Ctx) error {
	return c.JSON(Get())
}
