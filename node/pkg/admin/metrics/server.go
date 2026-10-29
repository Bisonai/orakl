package metrics

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// NewMux serves only GET /metrics (default registry, same as the admin
// /api/v1/metrics route); any other path is 404.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	return mux
}

// StartServer starts the metrics-only listener on METRICS_PORT, if set.
// The port is bound synchronously so a bind failure is returned to the
// caller (fail fast, like the admin listener); serving runs in the
// background for the process lifetime.
func StartServer() error {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		return nil
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("failed to bind metrics port %s: %w", port, err)
	}

	srv := &http.Server{
		Handler:           NewMux(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("metrics server stopped")
		}
	}()
	log.Info().Str("port", port).Msg("Metrics server started")
	return nil
}
