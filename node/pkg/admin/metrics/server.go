package metrics

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// handler serves the default registry; shared by the admin /api/v1/metrics
// route and the metrics-only listener so both expose the same families.
// Concurrent gathers are capped since the metrics port is unauthenticated.
var handler = promhttp.InstrumentMetricHandler(
	prometheus.DefaultRegisterer,
	promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		MaxRequestsInFlight: 10,
		Timeout:             10 * time.Second,
	}),
)

// NewMux serves only GET /metrics; any other path is 404.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handler)
	return mux
}

// StartServer binds the metrics-only listener on all interfaces and serves
// it in the background. The port is bound synchronously so a bind failure is
// returned to the caller (fail fast, like the admin listener).
func StartServer(port string) (*http.Server, error) {
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("invalid metrics port %q", port)
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind metrics port %s: %w", port, err)
	}
	log.Info().Str("port", port).Msg("Metrics server started")
	return Serve(ln), nil
}

// Serve serves the metrics mux on ln in the background.
func Serve(ln net.Listener) *http.Server {
	srv := &http.Server{
		Handler:           NewMux(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		// metrics are a side channel; never take the node down from here
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("metrics server stopped")
		}
	}()
	return srv
}
