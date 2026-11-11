package dal

import (
	"context"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/apiv2"
	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/configfetch"

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

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}
	collector.Start(ctx)

	hub := hub.HubSetup(ctx, configs)
	hub.Start(ctx, collector)

	startConfigReload(ctx, chain, len(configs), collector, hub)

	err = apiv2.Start(ctx, apiv2.WithCollector(collector), apiv2.WithHub(hub), apiv2.WithKeyCache(keyCache), apiv2.WithKeyStore(keyStore))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start DAL WS server")
		return err
	}

	return nil
}

// fetchConfigs fetches the chain's config bundle (retries, fallback source and
// optional cache: see configfetch).
func fetchConfigs(ctx context.Context, chain string) ([]Config, error) {
	return configfetch.Get[[]Config](ctx, configfetch.FeedsPath(chain))
}
