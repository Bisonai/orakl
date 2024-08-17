package dal

import (
	"context"
	"errors"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/apiv2"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/rs/zerolog/log"
)

type Config = types.Config

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	adminEndpoint := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if adminEndpoint == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL is not set")
	}

	configs, err := fetchConfigs(ctx, adminEndpoint)
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
	go hub.Start(ctx, collector)

	err = apiv2.Start(ctx, apiv2.WithCollector(collector), apiv2.WithHub(hub), apiv2.WithKeyCache(keyCache))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start DAL WS server")
		return err
	}

	return nil
}

func fetchConfigs(ctx context.Context, endpoint string) ([]Config, error) {
	return request.Request[[]Config](request.WithEndpoint(endpoint + "/config"))
}
