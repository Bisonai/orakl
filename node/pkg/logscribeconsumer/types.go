package logscribeconsumer

import (
	"time"

	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/rs/zerolog"
)

const (
	DefaultLogStoreInterval  = 10 * time.Second
	DefaultBufferSize        = 3000
	DefaultLogStoreLevel     = zerolog.ErrorLevel
	DefaultLogConsoleLevel   = zerolog.WarnLevel
	DefaultLogscribeEndpoint = "http://orakl-logscribe.orakl.svc.cluster.local:3000/api/v1/"
	ProcessLogsBatchSize     = 1000
)

type App struct {
	StoreInterval     time.Duration
	buffer            chan map[string]any
	consoleWriter     zerolog.ConsoleWriter
	LogscribeEndpoint string
	Service           string
	Level             zerolog.Level
	PostToLogscribe   bool
}

type AppConfig struct {
	StoreInterval     time.Duration
	Buffer            int
	LogscribeEndpoint string
	Service           string
	Level             string
	PostToLogscribe   bool
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
		c.LogscribeEndpoint = endpoint
	}
}

func WithStoreService(service string) AppOption {
	return func(c *AppConfig) {
		c.Service = service
	}
}

func WithStoreLevel(level string) AppOption {
	return func(c *AppConfig) {
		c.Level = level
	}
}

func WithPostToLogscribe(post bool) AppOption {
	return func(c *AppConfig) {
		c.PostToLogscribe = post
	}
}
