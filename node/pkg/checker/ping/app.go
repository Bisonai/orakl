package ping

import (
	"context"
	"os"
	"strconv"
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
)

var GlobalEndpoints = []string{
	"8.8.8.8",        // Google DNS
	"1.1.1.1",        // Cloudflare DNS
	"208.67.222.222", // OpenDNS
}

type PingerInterface interface {
	Run() error
	Statistics() *probing.Statistics
}

type PingResult struct {
	Address string
	Success bool
	Delay   time.Duration
}

type PingEndpoint struct {
	Address string
	Pinger  PingerInterface
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
}

func (pe *PingEndpoint) runOnce() error {
	// Type assert to actual *probing.Pinger to set Count=1
	if p, ok := pe.Pinger.(*probing.Pinger); ok {
		p.Count = 1
	}
	return pe.Pinger.Run()
}

func New(opts ...AppOption) (*App, error) {
	app := &App{
		RTTAvg:    make(map[string]float64),
		FailCount: make(map[string]int),
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

	for _, addr := range c.Endpoints {
		endpoint := addr // capture loop variable
		pinger, err := probing.NewPinger(endpoint)
		if err != nil {
			return nil, err
		}

		if !withoutPrivileged {
			pinger.SetPrivileged(true)
		}

		pinger.OnRecv = func(pkt *probing.Packet) {
			app.ResultsBuffer <- PingResult{
				Address: endpoint,
				Success: true,
				Delay:   pkt.Rtt,
			}
		}

		app.Endpoints = append(app.Endpoints, PingEndpoint{Address: endpoint, Pinger: pinger})
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
	// Start each endpoint's ping loop
	for _, e := range app.Endpoints {
		endpoint := e // capture for goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					log.Debug().Str("endpoint", endpoint.Address).Msg("sending ping")
					err := endpoint.runOnce()
					if err != nil {
						log.Warn().Err(err).Str("endpoint", endpoint.Address).Msg("ping error")
						app.ResultsBuffer <- PingResult{
							Address: endpoint.Address,
							Success: false,
							Delay:   0,
						}
					}
					time.Sleep(DefaultReconnectInterval)
				}
			}
		}()
	}

	// Process results
	for {
		select {
		case <-ctx.Done():
			close(app.ResultsBuffer)
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
	delayMs := result.Delay.Milliseconds()
	thresholdMs := int64(app.ThresholdFactor * app.RTTAvg[result.Address] / float64(time.Millisecond))

	if result.Success {
		if _, exists := app.RTTAvg[result.Address]; !exists {
			app.RTTAvg[result.Address] = float64(result.Delay)
		} else {
			app.RTTAvg[result.Address] = app.Alpha*float64(result.Delay) + (1-app.Alpha)*app.RTTAvg[result.Address]
		}

		if delayMs > thresholdMs {
			log.Warn().
				Str("endpoint", result.Address).
				Int64("delay_ms", delayMs).
				Int64("threshold_ms", thresholdMs).
				Msg("ping success but above threshold")
			app.FailCount[result.Address]++
		} else {
			log.Debug().Str("endpoint", result.Address).Int64("delay_ms", delayMs).Msg("ping OK")
			app.FailCount[result.Address] = 0
		}
	} else {
		log.Warn().Str("endpoint", result.Address).Msg("ping failed")
		app.FailCount[result.Address]++
	}
}
