package dal

import (
	"context"
	"fmt"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/apiv2"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/history"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/keystore"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/configfetch"

	"github.com/rs/zerolog/log"
)

type Config = types.Config

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	keyStore, err := keystore.OpenFromEnv(ctx, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open key store")
		return err
	}
	defer keyStore.Close()

	if n, countErr := keyStore.Count(ctx); countErr != nil {
		log.Error().Err(countErr).Msg("Failed to count API keys")
	} else if n == 0 {
		log.Warn().Msg("Key store has no API keys; all authenticated requests will be rejected")
	} else {
		log.Info().Int("keys", n).Msg("Loaded key store")
	}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(ctx, 10*time.Minute)

	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Error().Msg("CHAIN environment variable not set")
		return errorsentinel.ErrDalChainEnvNotFound
	}

	configs, err := fetchConfigs(ctx, chain)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch configs")
		return err
	}

	historyStore, err := openHistory(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set up history store")
		return err
	}
	if historyStore != nil {
		// deferred first, so it runs last: after the collector stopped, flush what is buffered
		defer func() {
			if err := historyStore.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close history store")
			}
		}()
	}

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}
	if historyStore != nil {
		collector.SetHistory(historyStore)
		historyStore.Start(ctx)
		// after the server returned, stop delivering before the history is closed
		defer collector.Stop()
	}
	collector.Start(ctx)

	hub := hub.HubSetup(ctx, configs)
	hub.Start(ctx, collector)

	startConfigReload(ctx, chain, len(configs), collector, hub)

	err = apiv2.Start(ctx, apiv2.WithCollector(collector), apiv2.WithHub(hub), apiv2.WithKeyCache(keyCache), apiv2.WithKeyStore(keyStore), apiv2.WithHistory(historyStore))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start DAL WS server")
		return err
	}

	return nil
}

// openHistory opens the price history store, or returns nil when history is
// disabled or, unless DAL_HISTORY_ENABLED is set, cannot be opened (DAL then
// runs without it). Invalid settings are an error.
func openHistory(ctx context.Context) (*history.Store, error) {
	cfg, enabled, err := history.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if !enabled {
		log.Info().Msg("Price history disabled")
		return nil, nil
	}
	store, err := history.Open(ctx, cfg)
	if err != nil {
		// explicitly enabled: fail loudly instead of serving 503 forever
		if os.Getenv(history.EnvEnabled) != "" {
			return nil, err
		}
		log.Error().Err(err).Msg("Failed to open history store; running without price history")
		return nil, nil
	}
	log.Info().Str("path", cfg.Path).Dur("interval", cfg.Interval).Dur("retention", cfg.Retention).Msg("Opened history store")
	return store, nil
}

// fetchConfigs fetches the chain's config bundle (retries and fallback source:
// see configfetch). An empty bundle (`[]`/`null`) is an error: starting with
// zero feeds would silently serve nothing.
func fetchConfigs(ctx context.Context, chain string) ([]Config, error) {
	path := configfetch.FeedsPath(chain)
	configs, err := configfetch.Get[[]Config](ctx, path)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("%s: %w", path, errorsentinel.ErrDalEmptyConfigs)
	}
	log.Info().Str("chain", chain).Int("count", len(configs)).Msg("Loaded configs")
	return configs, nil
}
