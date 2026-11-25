package health

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/secrets"
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

// checkTimeout bounds one target check so a hung target cannot stall the
// round (and freeze the health gauges).
var checkTimeout = 5 * time.Second

// selectHealthCheckJSON picks the embedded healthcheck bundle for the given
// chain. kairos keeps the baobab bundle, mainnet keeps the cypress bundle
// (the embedded healthcheck JSON filenames stay literal). The embedded
// file names stay literal on purpose.
func selectHealthCheckJSON(chain string) ([]byte, bool) {
	switch chain {
	case "kairos":
		return baobabJSON, true
	case "mainnet":
		return cypressJSON, true
	default:
		return nil, false
	}
}

func setUp() error {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "kairos"
	}
	HealthCheckInterval = 10 * time.Second

	interval := os.Getenv("HEALTH_CHECK_INTERVAL")
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse HEALTH_CHECK_INTERVAL, using default 10s")
	} else {
		HealthCheckInterval = parsedInterval
	}

	raw, ok := selectHealthCheckJSON(chain)
	if !ok {
		log.Error().Str("chain", chain).Msg("Invalid chain")
		return errors.New("invalid chain: " + chain)
	}
	if err = json.Unmarshal(raw, &HealthCheckUrls); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal healthcheck.json")
		return err
	}
	log.Info().Msg("Loaded healthcheck.json")

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

	for range ticker.C {
		for _, healthCheckUrl := range HealthCheckUrls {
			log.Debug().Str("name", healthCheckUrl.Name).Str("url", redactURL(healthCheckUrl.Url)).Msg("Checking health")
			recordHealth(healthCheckUrl.Name, checkUrl(ctx, healthCheckUrl))
		}
		recordRound(time.Now())
	}
	return nil
}

// redactURL masks the password of a URL for logging (SERVICE_DB_URL carries one).
func redactURL(raw string) string {
	u, err := neturl.Parse(raw)
	if err != nil {
		return "<unparsable url>"
	}
	return u.Redacted()
}

func checkUrl(ctx context.Context, healthCheckUrl HealthCheckUrl) bool {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	var alive bool
	if strings.HasPrefix(healthCheckUrl.Url, "http") {
		alive = checkHttp(ctx, healthCheckUrl.Url)
	} else if strings.HasPrefix(healthCheckUrl.Url, "redis") {
		alive = checkRedis(ctx, healthCheckUrl.Url)
	} else if strings.HasPrefix(healthCheckUrl.Url, "postgresql") {
		alive = checkPgs(ctx, healthCheckUrl.Url)
	}

	return alive
}

func checkHttp(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Info().Err(err).Msgf("Failed to check URL: %s", redactURL(url))
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Info().Err(err).Msgf("Failed to check URL: %s", redactURL(url))
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func checkRedis(ctx context.Context, url string) bool {
	url = strings.TrimPrefix(url, "redis://")
	// ContextTimeoutEnabled makes the client honor the check deadline
	// instead of only its own read/write timeouts.
	redisConnection := redis.NewClient(&redis.Options{Addr: url, ContextTimeoutEnabled: true})
	defer redisConnection.Close()
	_, err := redisConnection.Ping(ctx).Result()
	if err != nil {
		log.Info().Err(err).Msgf("Failed to check Redis URL: %s", url)
		return false
	}
	return true
}

func checkPgs(ctx context.Context, url string) bool {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Info().Err(err).Msgf("Failed to check PostgreSQL URL: %s", redactURL(url))
		return false
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Info().Err(err).Msgf("Failed to check PostgreSQL URL: %s", redactURL(url))
		return false
	}
	return true
}
