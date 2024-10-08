package apiv2

import (
	"net/http"

	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/stats"
)

type BulkResponse struct {
	Symbols        []string `json:"symbols"`
	Values         []string `json:"values"`
	AggregateTimes []string `json:"aggregateTimes"`
	Proofs         []string `json:"proofs"`
	FeedHashes     []string `json:"feedHashes"`
	Decimals       []string `json:"decimals"`
}

type ServerV2 struct {
	collector *collector.Collector
	hub       *hub.Hub
	keyCache  *keycache.KeyCache
	handler   http.Handler
}

type ServerV2Config struct {
	Port      string
	Collector *collector.Collector
	Hub       *hub.Hub
	KeyCache  *keycache.KeyCache
	StatsApp  *stats.StatsApp
}

type ServerV2Option func(*ServerV2Config)

func WithPort(port string) ServerV2Option {
	return func(config *ServerV2Config) {
		config.Port = port
	}
}

func WithCollector(c *collector.Collector) ServerV2Option {
	return func(config *ServerV2Config) {
		config.Collector = c
	}
}

func WithHub(h *hub.Hub) ServerV2Option {
	return func(config *ServerV2Config) {
		config.Hub = h
	}
}

func WithStatsApp(s *stats.StatsApp) ServerV2Option {
	return func(config *ServerV2Config) {
		config.StatsApp = s
	}
}

func WithKeyCache(k *keycache.KeyCache) ServerV2Option {
	return func(config *ServerV2Config) {
		config.KeyCache = k
	}
}
