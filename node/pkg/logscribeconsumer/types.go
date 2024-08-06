package logscribeconsumer

import (
	"time"

	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/rs/zerolog"
)

const (
	DefaultLogStoreInterval = 10 * time.Second
	DefaultBufferSize       = 3000
	MinimalLogStoreLevel    = zerolog.WarnLevel
	DefaultLogscribeEnpoint = "http://orakl-logscribe.orakl.svc.cluster.local:3000/api/v1/"
)

type App struct {
	StoreInterval    time.Duration
	buffer           chan map[string]any
	consoleWriter    zerolog.ConsoleWriter
	LogscribeEnpoint string
}

type AppConfig struct {
	StoreInterval    time.Duration
	Buffer           int
	LogscribeEnpoint string
}

type AppOption func(c *AppConfig)

type LogInsertModel api.LogInsertModel

func WithBuffer(buffer int) AppOption {
	return func(c *AppConfig) {
		if buffer <= 0 {
			buffer = DefaultBufferSize
		}
		c.Buffer = buffer
	}
}

func WithStoreInterval(interval time.Duration) AppOption {
	return func(c *AppConfig) {
		if interval <= 0 {
			interval = DefaultLogStoreInterval
		}
		c.StoreInterval = interval
	}
}

func WithLogscribeEndpoint(endpoint string) AppOption {
	return func(c *AppConfig) {
		c.LogscribeEnpoint = endpoint
	}
}
