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
	Alpha           float64 // smoothing factor for the EMA
	ThresholdFactor float64
	Endpoints       []PingEndpoint
	ResultsBuffer   chan PingResult
	FailCount       map[string]int
}

func (pe *PingEndpoint) run() error {
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

	endpoints := []PingEndpoint{}
	for _, endpoint := range c.Endpoints {
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

		endpoints = append(endpoints, PingEndpoint{endpoint, pinger})
	}

	app.Endpoints = endpoints
	app.Alpha = c.Alpha
	app.ThresholdFactor = c.ThresholdFactor

	return app, nil
}

func Run(ctx context.Context, opt ...AppOption) {
	app, err := New(opt...)
	if err != nil {
		panic(err)
	}
	app.Start(ctx)
}

func (app *App) Start(ctx context.Context) {
	for _, endpoint := range app.Endpoints {
		go func(endpoint PingEndpoint) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					log.Debug().Msg("connecting ICMP Pinger")
					err := endpoint.run()
					if err != nil {
						log.Error().Err(err).Msg("failed to ping endpoint")
						app.ResultsBuffer <- PingResult{
							Address: endpoint.Address,
							Success: false,
							Delay:   0,
						}
					}
					time.Sleep(DefaultReconnectInterval)
				}
			}
		}(endpoint)
	}

	for {
		select {
		case <-ctx.Done():
			close(app.ResultsBuffer)
			return
		case result := <-app.ResultsBuffer:
			if result.Success {
				if _, exists := app.RTTAvg[result.Address]; !exists {
					app.RTTAvg[result.Address] = float64(result.Delay)
				} else {
					app.RTTAvg[result.Address] = app.Alpha*float64(result.Delay) + (1-app.Alpha)*app.RTTAvg[result.Address]
				}

				if result.Delay > time.Duration(app.ThresholdFactor*app.RTTAvg[result.Address]) {
					log.Error().Any("result", result).Msg("ping failed")
					app.FailCount[result.Address] += 1
				} else {
					log.Debug().Any("result", result).Msg("ping success")
					app.FailCount[result.Address] = 0
				}
			} else {
				log.Error().Any("result", result).Msg("failed to ping endpoint")
				app.FailCount[result.Address] += 1
			}

			failedCount := 0
			for _, count := range app.FailCount {
				if count >= DefaultMaxFails {
					failedCount += 1
				}
			}

			// shuts down if all endpoints fails pinging 2 times in a row
			if failedCount == len(app.Endpoints) {
				log.Error().Msg("All pings failed, shutting down")
				return
			}
		}
	}
}
