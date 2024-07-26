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
		INSERT INTO configs (name, submit_interval) 
		VALUES (@name, @submit_interval) 
		ON CONFLICT (name) DO NOTHING
		RETURNING id, name, submit_interval;
	`
	DeleteConfigQuery = `DELETE FROM configs WHERE id = @id;`
)

func setup(ctx context.Context) (func() error, error) {
	submitInterval15000 := 15000
	submitInterval60000 := 60000

	configs := []Config{
		{
			Name:           "BTC-USDT",
			SubmitInterval: &submitInterval15000,
		},
		{
			Name:           "ETH-USDT",
			SubmitInterval: &submitInterval15000,
		},
		{
			Name:           "BTC-KRW",
			SubmitInterval: &submitInterval60000,
		},
		{
			Name:           "ETH-KRW",
			SubmitInterval: &submitInterval60000,
		},
	}

	var res []int32
	for _, config := range configs {
		insertedConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": config.Name, "submit_interval": config.SubmitInterval})
		if err != nil {
			log.Error().Err(err).Msgf("error inserting config %s", config.Name)
			return nil, err
		}
		res = append(res, insertedConfig.ID)
	}

	return cleanUp(ctx, res), nil
}

func cleanUp(ctx context.Context, configIds []int32) func() error {
	return func() error {
		for _, id := range configIds {
			_, err := db.QueryRow[any](ctx, DeleteConfigQuery, map[string]any{"id": id})
			if err != nil {
				log.Error().Err(err).Msgf("error deleting config %d", id)
				return err
			}
		}
		return nil
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}
