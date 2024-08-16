package offset

// track offset of latest local aggregate & global aggregate
// if offset exceeds threshold, will send an alarm

import (
	"context"
	"errors"
	"fmt"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/secrets"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	Threshold                = 5 * time.Second
	DefaultCheckInterval     = 5 * time.Minute
	GetLocalAggregateOffsets = `WITH latest_local_aggregate AS (
	SELECT
		la.config_id,
		MAX(la.timestamp) AS latest_timestamp
	FROM
		local_aggregates la
	GROUP BY
		la.config_id
),
aggregates_with_configs AS (
	SELECT
		c.name,
		lca.latest_timestamp
	FROM
		latest_local_aggregate lca
		JOIN configs c ON lca.config_id = c.id
)
SELECT
	awc.name,
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - awc.latest_timestamp)) AS delay_in_seconds
FROM
	aggregates_with_configs awc`
	GetGlobalAggregateOffsets = `WITH latest_global_aggregate AS (
	SELECT
		ga.config_id,
		MAX(ga.timestamp) AS latest_timestamp
	FROM
		global_aggregates ga
	GROUP BY
		ga.config_id
),
aggregates_with_configs AS (
	SELECT
		c.name,
		lga.latest_timestamp
	FROM
		latest_global_aggregate lga
		JOIN configs c ON lga.config_id = c.id
)
SELECT
	awc.name,
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - awc.latest_timestamp)) AS delay_in_seconds
FROM
	aggregates_with_configs awc`
)

type OffsetResult struct {
	Name  string  `db:"name"`
	Delay float64 `db:"delay_in_seconds"`
}

func Start(ctx context.Context) error {
	serviceDBUrl := secrets.GetSecret("SERVICE_DB_URL")
	if serviceDBUrl == "" {
		log.Error().Msg("Missing SERVICE_DB_URL")
		return errors.New("missing SERVICE_DB_URL")
	}

	serviceDB, err := db.GetTransientPool(ctx, serviceDBUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error getting service DB connection")
		return err
	}
	defer serviceDB.Close()

	ticker := time.NewTicker(DefaultCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context cancelled, shutting down")
			return ctx.Err()
		case <-ticker.C:
			err := checkOffsets(ctx, serviceDB)
			if err != nil {
				log.Error().Err(err).Msg("failed to get pgsql offset result")
			}
		}
	}

}

func checkOffsets(ctx context.Context, serviceDB *pgxpool.Pool) error {
	msg := ""

	localAggregateOffsetResults, err := db.QueryRowsTransient[OffsetResult](ctx, serviceDB, GetLocalAggregateOffsets, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting local aggregate offsets")
		return err
	}

	for _, result := range localAggregateOffsetResults {
		log.Debug().Str("name", result.Name).Float64("delay", result.Delay).Msg("local aggregate offset")
		if result.Delay > Threshold.Seconds() {
			msg += fmt.Sprintf("(local aggregate offset delayed) %s: %v seconds\n", result.Name, result.Delay)
		}
	}

	globalAggregateOffsetResults, err := db.QueryRowsTransient[OffsetResult](ctx, serviceDB, GetGlobalAggregateOffsets, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting global aggregate offsets")
		return err
	}

	for _, result := range globalAggregateOffsetResults {
		log.Debug().Str("name", result.Name).Float64("delay", result.Delay).Msg("global aggregate offset")
		if result.Delay > Threshold.Seconds() {
			msg += fmt.Sprintf("(global aggregate offset delayed) %s: %v seconds\n", result.Name, result.Delay)
		}
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}
