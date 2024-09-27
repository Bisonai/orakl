package dal

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/apiv2"
	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/stats"
	"bisonai.com/miko/node/pkg/utils/request"

	"github.com/rs/zerolog/log"
)

type Config = types.Config

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	statsApp := stats.Start(ctx)
	defer statsApp.Stop()

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	chain := os.Getenv("CHAIN")
	if chain == "" {
		return errors.New("CHAIN is not set")
	}

	symbols, err := fetchSymbols(chain)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch symbols")
		return err
	}

	collector, err := collector.NewCollector(ctx, symbols)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}
	collector.Start(ctx)

	hub := hub.HubSetup(ctx, symbols)
	go hub.Start(ctx, collector)

	err = apiv2.Start(ctx, apiv2.WithCollector(collector), apiv2.WithHub(hub), apiv2.WithKeyCache(keyCache), apiv2.WithStatsApp(statsApp))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start DAL WS server")
		return err
	}

	return nil
}

func fetchSymbols(chain string) ([]string, error) {
	type ConfigEntry struct {
		Name string `json:"name"`
	}

	url := "https://config.orakl.network/" + strings.ToLower(chain) + "_configs.json"

	results, err := request.Request[[]ConfigEntry](request.WithEndpoint(url), request.WithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}

	var symbols []string

	for _, result := range results {
		symbols = append(symbols, result.Name)
	}

	return symbols, nil
}
