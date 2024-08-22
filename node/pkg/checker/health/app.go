package health

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/alert"
	"bisonai.com/miko/node/pkg/secrets"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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

func setUp() error {
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
		err = json.Unmarshal(baobabJSON, &HealthCheckUrls)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal baobab_healthcheck.json")
			return err
		}
	} else if chain == "cypress" {
		err = json.Unmarshal(cypressJSON, &HealthCheckUrls)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal cypress_healthcheck.json")
			return err
		}
	} else {
		log.Error().Msg("Invalid chain")
		return err
	}
	log.Info().Msg("Loaded healthcheck.json")

	graphnodeDB := secrets.GetSecret("DATABASE_URL")
	if graphnodeDB != "" {
		HealthCheckUrls = append(HealthCheckUrls, HealthCheckUrl{
			Name: "graphnode",
			Url:  graphnodeDB,
		})
	}

	log.Info().Msg("Loaded graphnode db url for healthcheck")

	serviceDB := secrets.GetSecret("SERVICE_DB_URL")
	if serviceDB != "" {
		HealthCheckUrls = append(HealthCheckUrls, HealthCheckUrl{
			Name: "service",
			Url:  serviceDB,
		})
	}

	log.Info().Msg("Loaded service db url for healthcheck")

	return nil
}

func Start(ctx context.Context) error {
	err := setUp()
	if err != nil {
		return err
	}

	log.Info().Msg("Starting health checker")
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	downServices := make(map[string]bool)

	for range ticker.C {
		alarmMessage := ""
		for _, healthCheckUrl := range HealthCheckUrls {
			log.Debug().Str("name", healthCheckUrl.Name).Str("url", healthCheckUrl.Url).Msg("Checking health")
			isAlive := checkUrl(ctx, healthCheckUrl)
			if !isAlive {
				downServices[healthCheckUrl.Name] = true
				alarmMessage += healthCheckUrl.Name + " is down\n"
			} else if serviceWasDown, serviceNameFound := downServices[healthCheckUrl.Name]; serviceWasDown && serviceNameFound {
				downServices[healthCheckUrl.Name] = false
				alarmMessage += healthCheckUrl.Name + " is back up\n"
			}
		}
		if alarmMessage != "" {
			alert.SlackAlert(alarmMessage)
		}
	}
	return nil
}

func checkUrl(ctx context.Context, healthCheckUrl HealthCheckUrl) bool {
	var alive bool
	if strings.HasPrefix(healthCheckUrl.Url, "http") {
		alive = checkHttp(healthCheckUrl.Url)
	} else if strings.HasPrefix(healthCheckUrl.Url, "redis") {
		alive = checkRedis(ctx, healthCheckUrl.Url)
	} else if strings.HasPrefix(healthCheckUrl.Url, "postgresql") {
		alive = checkPgs(ctx, healthCheckUrl.Url)
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

func checkPgs(ctx context.Context, url string) bool {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check PostgreSQL URL: %s", url)
		return false
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check PostgreSQL URL: %s", url)
		return false
	}
	return true
}
