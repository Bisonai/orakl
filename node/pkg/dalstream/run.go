package dalstream

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/utils/buildinfo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const shutdownTimeout = 5 * time.Second

// Run serves until ctx is done, then closes all streams and returns.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	// cancelled also when a listener fails, to stop streams, hub and upstream
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	hub := NewHub(cfg)
	unregister, err := registerUpstreamAge(prometheus.DefaultRegisterer, hub)
	if err != nil {
		return err
	}
	defer unregister()

	publicLn, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	metricsLn, err := net.Listen("tcp", cfg.MetricsAddr)
	if err != nil {
		publicLn.Close()
		return err
	}

	public := &http.Server{
		Handler:           NewServer(hub, cfg),
		ReadHeaderTimeout: 10 * time.Second,
		// streams end when ctx is done, so Shutdown does not wait on them
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())
	metricsMux.HandleFunc("GET /version", buildinfo.HTTPHandler)
	metrics := &http.Server{Handler: metricsMux, ReadHeaderTimeout: 10 * time.Second}

	var wg sync.WaitGroup
	errc := make(chan error, 2)
	serve := func(srv *http.Server, ln net.Listener) {
		defer wg.Done()
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}
	wg.Add(4)
	go serve(public, publicLn)
	go serve(metrics, metricsLn)
	go func() { defer wg.Done(); hub.Run(ctx, cfg.Tick) }()
	go func() { defer wg.Done(); NewUpstream(cfg, hub).Run(ctx) }()
	log.Info().Str("addr", cfg.Addr).Str("metricsAddr", cfg.MetricsAddr).Dur("tick", cfg.Tick).Msg("dal-stream started")

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errc:
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	for _, srv := range []*http.Server{public, metrics} {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Warn().Err(err).Msg("dal-stream shutdown")
		}
	}
	wg.Wait()
	return runErr
}
