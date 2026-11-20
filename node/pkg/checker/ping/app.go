package ping

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog/log"
)

const (
	DefaultAlpha             = 0.3
	DefaultThresholdFactor   = 1.3
	DefaultMaxFails          = 2
	DefaultBufferSize        = 500
	DefaultReconnectInterval = 2 * time.Second
	// DefaultTimeout bounds a single ping attempt; a reply that does not arrive
	// within it counts as a failure.
	DefaultTimeout = 3 * time.Second
	// DefaultMinThresholdMargin is the least an RTT must exceed the average by
	// to count as slow, so sub-millisecond jitter is not a failure.
	DefaultMinThresholdMargin = time.Millisecond
)

var GlobalEndpoints = []string{
	"8.8.8.8",        // Google DNS
	"1.1.1.1",        // Cloudflare DNS
	"208.67.222.222", // OpenDNS
}

type PingerInterface interface {
	RunWithContext(ctx context.Context) error
	Statistics() *probing.Statistics
}

// pingerFactory builds a fresh pinger for one attempt. probing.Pinger is
// single-use: Stop closes its done channel and statistics accumulate, so it
// must not be reused across attempts.
type pingerFactory func(addr string, timeout time.Duration) (PingerInterface, error)

type PingResult struct {
	Address string
	Success bool
	Delay   time.Duration
}

type PingEndpoint struct {
	Address string
}

type AppConfig struct {
	Endpoints       []string
	BufferSize      int
	Alpha           float64
	ThresholdFactor float64
}

type AppOption func(*AppConfig)

func WithEndpoints(endpoints []string) AppOption {
	return func(c *AppConfig) {
		c.Endpoints = endpoints
	}
}

func WithResultBuffer(size int) AppOption {
	return func(c *AppConfig) {
		c.BufferSize = size
	}
}

func WithAlpha(alpha float64) AppOption {
	return func(c *AppConfig) {
		c.Alpha = alpha
	}
}

func WithThresholdFactor(factor float64) AppOption {
	return func(c *AppConfig) {
		c.ThresholdFactor = factor
	}
}

type App struct {
	RTTAvg          map[string]float64
	Alpha           float64
	ThresholdFactor float64
	Endpoints       []PingEndpoint
	ResultsBuffer   chan PingResult
	FailCount       map[string]int

	timeout   time.Duration
	interval  time.Duration
	newPinger pingerFactory
}

func newProbingPinger(privileged bool) pingerFactory {
	return func(addr string, timeout time.Duration) (PingerInterface, error) {
		pinger, err := probing.NewPinger(addr)
		if err != nil {
			return nil, err
		}
		pinger.Count = 1
		pinger.Timeout = timeout
		pinger.SetPrivileged(privileged)
		return pinger, nil
	}
}

// runOnce sends one ping and returns its result. ok is false when ctx was
// canceled, in which case no result should be reported.
func (app *App) runOnce(ctx context.Context, addr string) (result PingResult, ok bool) {
	failure := PingResult{Address: addr, Success: false}

	pinger, err := app.newPinger(addr, app.timeout)
	if err != nil {
		log.Warn().Err(err).Str("endpoint", addr).Msg("ping setup error")
		return failure, ctx.Err() == nil
	}

	runCtx, cancel := context.WithTimeout(ctx, app.timeout)
	defer cancel()
	err = pinger.RunWithContext(runCtx)
	if ctx.Err() != nil {
		return PingResult{}, false
	}

	stats := pinger.Statistics()
	if stats == nil || stats.PacketsRecv == 0 {
		log.Warn().Err(err).Str("endpoint", addr).Dur("timeout", app.timeout).Msg("no ping reply")
		return failure, true
	}
	return PingResult{Address: addr, Success: true, Delay: stats.AvgRtt}, true
}

func New(opts ...AppOption) (*App, error) {
	app := &App{
		RTTAvg:    make(map[string]float64),
		FailCount: make(map[string]int),
		timeout:   DefaultTimeout,
		interval:  DefaultReconnectInterval,
	}

	c := AppConfig{
		Endpoints:       GlobalEndpoints,
		BufferSize:      DefaultBufferSize,
		Alpha:           DefaultAlpha,
		ThresholdFactor: DefaultThresholdFactor,
	}

	for _, opt := range opts {
		opt(&c)
	}

	app.ResultsBuffer = make(chan PingResult, c.BufferSize)

	withoutPrivileged, err := strconv.ParseBool(os.Getenv("WITHOUT_PING_PRIVILEGED"))
	if err != nil {
		withoutPrivileged = false
	}

	app.newPinger = newProbingPinger(!withoutPrivileged)

	for _, addr := range c.Endpoints {
		// Validate the address up front, as before.
		if _, err := probing.NewPinger(addr); err != nil {
			return nil, err
		}
		app.Endpoints = append(app.Endpoints, PingEndpoint{Address: addr})
	}

	app.Alpha = c.Alpha
	app.ThresholdFactor = c.ThresholdFactor

	return app, nil
}

func Run(ctx context.Context, opts ...AppOption) {
	app, err := New(opts...)
	if err != nil {
		panic(err)
	}
	app.Start(ctx)
}

func (app *App) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	// Stop the ping loops and wait for them on return, so nothing sends on
	// ResultsBuffer afterwards. ResultsBuffer is never closed.
	defer func() {
		cancel()
		wg.Wait()
	}()

	// Start each endpoint's ping loop
	for _, e := range app.Endpoints {
		addr := e.Address
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				log.Debug().Str("endpoint", addr).Msg("sending ping")
				result, ok := app.runOnce(ctx, addr)
				if !ok {
					return
				}
				select {
				case app.ResultsBuffer <- result:
				case <-ctx.Done():
					return
				}
				select {
				case <-time.After(app.interval):
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Process results
	for {
		select {
		case <-ctx.Done():
			return
		case result := <-app.ResultsBuffer:
			app.processResult(result)

			// Check shutdown condition
			failedCount := 0
			for _, count := range app.FailCount {
				if count >= DefaultMaxFails {
					failedCount++
				}
			}

			if failedCount == len(app.Endpoints) {
				log.Error().Msg("All endpoints failed multiple times, shutting down")
				return
			}
		}
	}
}

func (app *App) processResult(result PingResult) {
	if !result.Success {
		log.Warn().Str("endpoint", result.Address).Msg("ping failed")
		app.FailCount[result.Address]++
		return
	}

	avg, seen := app.RTTAvg[result.Address]
	if !seen {
		// The first sample only initialises the average.
		app.RTTAvg[result.Address] = float64(result.Delay)
		app.FailCount[result.Address] = 0
		log.Debug().Str("endpoint", result.Address).Dur("delay", result.Delay).Msg("ping OK")
		return
	}

	threshold := max(time.Duration(app.ThresholdFactor*avg), time.Duration(avg)+DefaultMinThresholdMargin)
	app.RTTAvg[result.Address] = app.Alpha*float64(result.Delay) + (1-app.Alpha)*avg

	if result.Delay > threshold {
		log.Warn().
			Str("endpoint", result.Address).
			Dur("delay", result.Delay).
			Dur("threshold", threshold).
			Msg("ping success but above threshold")
		app.FailCount[result.Address]++
	} else {
		log.Debug().Str("endpoint", result.Address).Dur("delay", result.Delay).Msg("ping OK")
		app.FailCount[result.Address] = 0
	}
}
