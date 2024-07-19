package lograkl

import (
	"time"
)

const DefaultLogStoreInterval = 5 * time.Second

type App struct {
	StoreInterval time.Duration
	buffer        chan map[string]any
}

type AppConfig struct {
	StoreInterval time.Duration
	Buffer        int
}

type AppOption func(c *AppConfig)

func WithBuffer(buffer int) AppOption {
	return func(c *AppConfig) {
		c.Buffer = buffer
	}
}

func WithStoreInterval(interval time.Duration) AppOption {
	return func(c *AppConfig) {
		c.StoreInterval = interval
	}
}
