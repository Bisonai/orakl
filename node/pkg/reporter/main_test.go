//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	InsertConfigQuery = `
		INSERT INTO configs (name, aggregate_interval, submit_interval) 
		VALUES (@name, @aggregate_interval, @submit_interval) 
		ON CONFLICT (name) DO NOTHING
		RETURNING name, id, submit_interval, aggregate_interval;
	`
	DeleteConfigQuery = `DELETE FROM configs WHERE id = @id;`
)

func setup(ctx context.Context) ([]int32, error) {
	aggregateInterval := 400
	submitInterval15000 := 15000
	submitInterval60000 := 60000

	configs := []Config{
		{
			Name:              "BTC-USDT",
			AggregateInterval: &aggregateInterval,
			SubmitInterval:    &submitInterval15000,
		},
		{
			Name:              "ETH-USDT",
			AggregateInterval: &aggregateInterval,
			SubmitInterval:    &submitInterval15000,
		},
		{
			Name:              "BTC-KRW",
			AggregateInterval: &aggregateInterval,
			SubmitInterval:    &submitInterval60000,
		},
		{
			Name:              "ETH-KRW",
			AggregateInterval: &aggregateInterval,
			SubmitInterval:    &submitInterval60000,
		},
	}

	var res []int32
	for _, config := range configs {
		insertedConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": config.Name, "aggregate_interval": config.AggregateInterval, "submit_interval": config.SubmitInterval})
		if err != nil {
			log.Error().Err(err).Msgf("error inserting config %s", config.Name)
			return nil, err
		}
		res = append(res, insertedConfig.ID)
	}

	return res, nil
}

func cleanUp(ctx context.Context, configIds []int32) error {
	for _, id := range configIds {
		_, err := db.QueryRow[any](ctx, DeleteConfigQuery, map[string]any{"id": id})
		if err != nil {
			log.Error().Err(err).Msgf("error deleting config %d", id)
			return err
		}
	}

	db.ClosePool()

	return nil
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	configIds, err := setup(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error setting up test")
		os.Exit(1)
	}

	code := m.Run()

	err = cleanUp(ctx, configIds)
	if err != nil {
		log.Fatal().Err(err).Msg("error cleaning up configs")
		os.Exit(1)
	}
	os.Exit(code)
}
