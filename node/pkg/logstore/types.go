package logstore

import "time"

const DefaultLogStoreInterval = 5 * time.Second

type LogStore struct {
	StoreInterval time.Duration
	logChannel    chan []byte
	logEntries    [][]byte
}

type LogStoreConfig struct {
	StoreInterval time.Duration
	Buffer        int
}

type LogStoreOption func(c *LogStoreConfig)

func WithBuffer(buffer int) LogStoreOption {
	return func(c *LogStoreConfig) {
		c.Buffer = buffer
	}
}

func WithStoreInterval(interval time.Duration) LogStoreOption {
	return func(c *LogStoreConfig) {
		c.StoreInterval = interval
	}
}
