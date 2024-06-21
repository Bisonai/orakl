package health

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"github.com/rs/zerolog/log"
)

type HealthCheckUrl struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

//go:embed baobab_healthcheck.json
var baobabJSON []byte

//go:embed cypress_healthcheck.json
var cypressJSON []byte

var HealthCheckUrls []HealthCheckUrl
var HealthCheckInterval time.Duration

func init() {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}
	HealthCheckInterval = 10 * time.Second

	interval := os.Getenv("HEALTH_CHECK_INTERVAL")
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse HEALTH_CHECK_INTERVAL, using default 10s")
	} else {
		HealthCheckInterval = parsedInterval
	}

	if chain == "baobab" {
		err := json.Unmarshal(baobabJSON, &HealthCheckUrls)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal baobab_healthcheck.json")
			os.Exit(1)
		}
	} else if chain == "cypress" {
		err := json.Unmarshal(cypressJSON, &HealthCheckUrls)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal cypress_healthcheck.json")
			os.Exit(1)
		}
	} else {
		log.Error().Msg("Invalid chain")
		os.Exit(1)
	}
	log.Info().Msg("Loaded healthcheck.json")
}

func Start() {
	log.Info().Msg("Starting health checker")
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		alarmMessage := ""
		for _, healthCheckUrl := range HealthCheckUrls {
			log.Debug().Str("name", healthCheckUrl.Name).Str("url", healthCheckUrl.Url).Msg("Checking health")
			isAlive := checkUrl(healthCheckUrl)
			if !isAlive {
				alarmMessage += healthCheckUrl.Name + " is down\n"
			}
		}
		if alarmMessage != "" {
			alert.SlackAlert(alarmMessage)
		}
	}
}

func checkUrl(healthCheckUrl HealthCheckUrl) bool {
	var alive bool
	if strings.HasPrefix(healthCheckUrl.Url, "http") {
		alive = checkHttp(healthCheckUrl.Url)
	} else if strings.HasPrefix(healthCheckUrl.Url, "redis") {
		alive = checkRedis(context.Background(), healthCheckUrl.Url)
	}

	return alive
}

func checkHttp(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check URL: %s", url)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func checkRedis(ctx context.Context, url string) bool {
	url = strings.TrimPrefix(url, "redis://")
	redisConnection := redis.NewClient(&redis.Options{Addr: url})
	defer redisConnection.Close()
	_, err := redisConnection.Ping(ctx).Result()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check Redis URL: %s", url)
		return false
	}
	return true
}
