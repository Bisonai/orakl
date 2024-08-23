package ping

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog/log"
)

const (
	DefaultPingerTimeout     = 2 * time.Second
	DefaultMaxDelay          = 100 * time.Millisecond
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
	MaxDelay   time.Duration
	Endpoints  []string
	BufferSize int
}

type AppOption func(*AppConfig)

func WithMaxDelay(duration time.Duration) AppOption {
	return func(c *AppConfig) {
		c.MaxDelay = duration
	}
}

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

type App struct {
	MaxDelay      time.Duration
	Endpoints     []PingEndpoint
	FailCount     map[string]int
	ResultsBuffer chan PingResult
}

func (pe *PingEndpoint) run() error {
	return pe.Pinger.Run()
}

func New(opts ...AppOption) (*App, error) {
	app := &App{}

	c := AppConfig{
		MaxDelay:   DefaultMaxDelay,
		Endpoints:  GlobalEndpoints,
		BufferSize: DefaultBufferSize,
	}

	for _, opt := range opts {
		opt(&c)
	}

	app.ResultsBuffer = make(chan PingResult, c.BufferSize)

	endpoints := []PingEndpoint{}
	for _, endpoint := range c.Endpoints {
		pinger, err := probing.NewPinger(endpoint)
		if err != nil {
			return nil, err
		}

		pinger.Timeout = DefaultPingerTimeout
		pinger.Count = 0

		pinger.OnRecv = func(pkt *probing.Packet) {
			app.ResultsBuffer <- PingResult{
				Address: endpoint,
				Success: true,
				Delay:   pkt.Rtt,
			}
		}

		endpoints = append(endpoints, PingEndpoint{endpoint, pinger})
	}

	app.MaxDelay = c.MaxDelay
	app.Endpoints = endpoints
	app.FailCount = make(map[string]int)

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
						app.ResultsBuffer <- PingResult{
							Address: endpoint.Address,
							Success: false,
							Delay:   0,
						}
					}
					time.Sleep(DefaultReconnectInterval)
					continue
				}
			}
		}(endpoint)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case result := <-app.ResultsBuffer:
			if result.Success && result.Delay < app.MaxDelay {
				app.FailCount[result.Address] = 0
			} else {
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
