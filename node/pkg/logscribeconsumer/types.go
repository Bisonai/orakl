package logscribeconsumer

import (
	"time"

	"github.com/rs/zerolog"
)

const (
	DefaultLogStoreInterval = 10 * time.Second
	DefaultBufferSize       = 3000
	MinimalLogStoreLevel    = zerolog.WarnLevel
)

type App struct {
	StoreInterval time.Duration
	buffer        chan map[string]any
	consoleWriter zerolog.ConsoleWriter
}

type AppConfig struct {
	StoreInterval time.Duration
	Buffer        int
}

type AppOption func(c *AppConfig)

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
