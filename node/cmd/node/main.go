package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"bisonai.com/miko/node/pkg/admin"
	adminMetrics "bisonai.com/miko/node/pkg/admin/metrics"
	adminUtils "bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/checker/ping"
	"bisonai.com/miko/node/pkg/fetcher"
	"bisonai.com/miko/node/pkg/libp2p/helper"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/utils/loginit"
	"bisonai.com/miko/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	if len(os.Args) > 1 && os.Args[1] == "state" {
		if err := runState(ctx, os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "admin" {
		if err := runAdmin(ctx, os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(adminExitCode(err))
		}
		return
	}

	loginit.InitZeroLog()

	// fail before joining the network if ADMIN_AUTH_REQUIRED can't be met;
	// admin.Run checks again for the other binaries
	if _, err := adminUtils.AdminToken(); err != nil {
		log.Error().Err(err).Msg("Failed to resolve admin token")
		os.Exit(1)
	}

	// optional metrics-only listener (GET /metrics), separate from the admin API
	if port := os.Getenv("METRICS_PORT"); port != "" {
		mp, _ := strconv.Atoi(port)
		ap, _ := strconv.Atoi(adminPort())
		if mp != 0 && mp == ap {
			log.Error().Str("port", port).Msg("METRICS_PORT must differ from APP_PORT")
			os.Exit(1)
		}
		if _, err := adminMetrics.StartServer(port); err != nil {
			log.Error().Err(err).Msg("Failed to start metrics server")
			os.Exit(1)
		}
	}

	// signer keyring + latest rounds; a missing file (wrong path, unmounted
	// volume) is fatal so the node never starts on an empty keyring/rounds
	state, err := nodestate.OpenFromEnv(ctx, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open node state")
		fmt.Fprintln(os.Stderr, "failed to open node state:", err)
		os.Exit(1)
	}
	defer state.Close()

	mb := bus.New(1000)
	var wg sync.WaitGroup

	host, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
	} else {
		libp2pSetup.RegisterPeerMetrics(host)
	}

	ps, err := libp2pSetup.MakePubsub(ctx, host)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make pubsub")
	}

	err = retrier.Retry(func() error {
		return libp2pSetup.ConnectThroughBootApi(ctx, host)
	}, 5, 10*time.Second, 30*time.Second)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup libp2p")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Any("panic", r).Msg("panic recovered from network checks")
			}
		}()
		time.Sleep(10 * time.Second) // give some buffer until the app is ready
		ping.Run(ctx)
		os.Exit(1)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		adminErr := admin.Run(ctx, mb)
		if adminErr != nil {
			log.Error().Err(adminErr).Msg("Failed to start admin server")
			os.Exit(1)
		}
	}()

	log.Info().Msg("Syncing orakl config")
	err = admin.SyncMikoConfig(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync orakl config")
		return
	}
	log.Info().Msg("Orakl config synced")

	wg.Add(1)
	go func() {
		defer wg.Done()
		f := fetcher.New(mb)
		fetcherErr := f.Run(ctx)
		if fetcherErr != nil {
			log.Error().Err(fetcherErr).Msg("Failed to start fetcher")
			os.Exit(1)
		}
	}()
	log.Info().Msg("Fetcher started")

	a := aggregator.New(mb, host, ps, state)
	// cancelled on shutdown: every aggregator stops and resigns
	aggregatorCtx, stopAggregators := context.WithCancel(ctx)
	go handOverOnSignal(shutdownSteps{
		stopAggregators: stopAggregators,
		waitAggregators: a.WaitStopped,
		closeHost: func() error {
			if host == nil {
				return nil
			}
			return host.Close()
		},
		flushRounds: a.FlushRounds,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		aggregatorErr := a.Run(aggregatorCtx)
		// a shutdown during startup fails Run; the hand-over exits
		if aggregatorErr != nil && aggregatorCtx.Err() == nil {
			log.Error().Err(aggregatorErr).Msg("Failed to start aggregator")
			os.Exit(1)
		}
	}()
	log.Info().Msg("Aggregator started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		helperApp := helper.New(mb, host)
		libp2pHelperErr := helperApp.Run(ctx)
		if libp2pHelperErr != nil {
			log.Error().Err(libp2pHelperErr).Msg("Failed to start libp2p helper")
			os.Exit(1)
		}
	}()
	log.Info().Msg("libp2p helper started")

	wg.Wait()
}

// Per-step bounds of the shutdown hand-over; 13 s in all, well under the
// pod's 30 s grace period. Variables for tests.
var (
	resignTimeout    = 5 * time.Second
	closeHostTimeout = 3 * time.Second
	flushTimeout     = 5 * time.Second
)

// shutdownSteps are what handOver runs, in order.
type shutdownSteps struct {
	// cancels every aggregator's Run: each feed's Raft resigns the leader role
	stopAggregators func()
	// until every aggregator's Run returned, so none produces a round after
	waitAggregators func(context.Context) error
	// peers see the disconnect and elect at once, for any feed whose
	// unsubscribe didn't reach them first
	closeHost   func() error
	flushRounds func(context.Context) error
}

// handOverOnSignal hands over and exits on SIGTERM/SIGINT.
func handOverOnSignal(steps shutdownSteps) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	log.Info().Str("signal", sig.String()).Msg("shutting down, handing over")
	go func() {
		sig := <-sigCh
		log.Warn().Str("signal", sig.String()).Msg("second signal, exiting without hand-over")
		os.Exit(1)
	}()
	handOver(steps)
	// no state.Close: other goroutines may still write; SQLite (WAL) is
	// consistent on exit, and a round advanced after the flush is covered by
	// the stored+2 restart gap
	os.Exit(0)
}

// handOver gives up every feed this node leads, so peers take over within
// about a second instead of timing it out: it stops the aggregators (each
// resigns, and its topic unsubscribe tells peers at once), waits until none can
// produce a round, closes the libp2p host so peers also see the disconnect,
// then persists rounds so the next start continues after the last used one.
// Each step is bounded and the next runs regardless. The node's network path
// must outlive this, or nothing reaches the peers.
func handOver(steps shutdownSteps) {
	start := time.Now()

	steps.stopAggregators()
	ctx, cancel := context.WithTimeout(context.Background(), resignTimeout)
	if err := steps.waitAggregators(ctx); err != nil {
		log.Error().Err(err).Msg("aggregators still running at shutdown")
	}
	cancel()
	resigned := time.Since(start)

	closed := make(chan error, 1)
	go func() { closed <- steps.closeHost() }()
	select {
	case err := <-closed:
		if err != nil {
			log.Error().Err(err).Msg("failed to close libp2p host on shutdown")
		}
	case <-time.After(closeHostTimeout):
		log.Error().Dur("timeout", closeHostTimeout).Msg("libp2p host not closed on shutdown")
	}
	hostClosed := time.Since(start)

	ctx, cancel = context.WithTimeout(context.Background(), flushTimeout)
	if err := steps.flushRounds(ctx); err != nil {
		log.Error().Err(err).Msg("failed to persist rounds on shutdown")
	}
	cancel()

	// a log, not a metric: the process exits before another scrape
	log.Info().
		Dur("resigned", resigned).
		Dur("hostClosed", hostClosed).
		Dur("total", time.Since(start)).
		Msg("shutdown hand-over done")
}
