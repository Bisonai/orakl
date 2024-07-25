//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const InsertConfigQuery = `INSERT INTO configs (name, aggregate_interval, submit_interval) VALUES (@name, @aggregate_interval, @submit_interval) RETURNING name, id, submit_interval, aggregate_interval;`

func getConfigUrl() string {
	configUrl := os.Getenv("ADMIN_ENDPOINT")
	if configUrl == "" {
		configUrl = "http://100.65.92.37:3030/api/v1"
	}

	return configUrl + "/config"
}

func fetchConfigs(ctx context.Context) ([]Config, error) {
	configUrl := getConfigUrl()
	configs, err := request.Request[[]Config](request.WithEndpoint(configUrl))
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func setup(ctx context.Context) error {
	err := db.QueryWithoutResult(ctx, "DELETE FROM configs;", nil)
	if err != nil {
		return err
	}

	configs, err := fetchConfigs(ctx)
	if err != nil {
		return err
	}

	for _, config := range configs {
		_, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": config.Name, "aggregate_interval": config.AggregateInterval, "submit_interval": config.SubmitInterval})
		if err != nil {
			log.Error().Err(err).Msgf("error inserting config %s", config.Name)
			return err
		}
	}

	return nil
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	err := setup(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up test")
		os.Exit(1)
	}

	code := m.Run()

	db.ClosePool()
	os.Exit(code)
}
