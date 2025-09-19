package offset

// track offset of latest local aggregate & global aggregate
// if offset exceeds threshold, will send an alarm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"bisonai.com/miko/node/pkg/alert"
	"bisonai.com/miko/node/pkg/checker"
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/secrets"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	AlarmOffsetPerPair = 10
	AlarmOffsetInTotal = 3

	PriceDifferenceThreshold = 0.1
	DelayThreshold           = 5 * time.Second
	DefaultCheckInterval     = 5 * time.Minute

	GetSingleLocalAggregateOffset = `SELECT
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - timestamp)) AS delay_in_seconds
FROM
	local_aggregates
WHERE
	config_id = '%d'
ORDER BY timestamp DESC
LIMIT 1`

	GetSingleGlobalAggregateOffset = `SELECT
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - timestamp)) AS delay_in_seconds
FROM
	global_aggregates
WHERE
	config_id = '%d'
ORDER BY timestamp DESC
LIMIT 1`

	GetSingleLatestAggregates = `WITH latest_local_aggregate AS (
	SELECT
		la.config_id,
		la.value AS local_aggregate
	FROM
		local_aggregates la
	WHERE
		la.config_id = '%d'
	ORDER BY
		la.timestamp DESC
	LIMIT 1
),
latest_global_aggregate AS (
	SELECT
	 ga.config_id,
	 ga.value AS global_aggregate
	FROM
		global_aggregates ga
	WHERE
		ga.config_id = '%d'
	ORDER BY ga.timestamp DESC
	LIMIT 1
)
SELECT
 la.local_aggregate,
 ga.global_aggregate
FROM
 latest_local_aggregate la
 JOIN latest_global_aggregate ga on la.config_id = ga.config_id`

	PorAlarmThreshold     = 3
	PorAlarmOffset        = 300 * time.Second
	GetPorOffchainOffsets = `SELECT
	name,
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - MAX(timestamp))) AS delay
FROM
	por_offchain
GROUP BY
	name`
)

type Config struct {
	ID   int32  `db:"id"`
	Name string `db:"name"`
}

type OffsetResultEach struct {
	Delay float64 `db:"delay_in_seconds"`
}

type LatestAggregateEach struct {
	LocalAggregate  float64 `db:"local_aggregate"`
	GlobalAggregate float64 `db:"global_aggregate"`
}

type PorOffsetResult struct {
	Name  string
	Delay float64
}

var localAggregateAlarmCount map[int32]int
var globalAggregateAlarmCount map[int32]int
var porOffchainAlarmCount map[string]int

func Start(ctx context.Context) error {
	localAggregateAlarmCount = make(map[int32]int)
	globalAggregateAlarmCount = make(map[int32]int)
	porOffchainAlarmCount = make(map[string]int)

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
			go checkOffsets(ctx, serviceDB)
			go checkPorOffsets(ctx, serviceDB)
		}
	}
}

func checkPorOffsets(ctx context.Context, serviceDB *pgxpool.Pool) {
	msg := ""

	res, err := db.QueryRowsTransient[PorOffsetResult](ctx, serviceDB, GetPorOffchainOffsets, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting por offchain offsets")
		return
	}

	for _, r := range res {
		log.Debug().Str("name", r.Name).Float64("delay", r.Delay).Msg("checking por offset")
		if r.Delay > PorAlarmOffset.Seconds() {
			porOffchainAlarmCount[r.Name]++
			if porOffchainAlarmCount[r.Name] > PorAlarmThreshold {
				msg += fmt.Sprintf("(por offchain offset) %s: %v seconds\n", r.Name, r.Delay)
				porOffchainAlarmCount[r.Name] = 0
			}
		} else {
			porOffchainAlarmCount[r.Name] = 0
		}
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func checkOffsets(ctx context.Context, serviceDB *pgxpool.Pool) {
	msg := ""

	loadedConfigs, err := db.QueryRowsTransient[Config](ctx, serviceDB, "SELECT id, name FROM configs", nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting configs")
		return
	}

	localAggregateDelayedOffsetCount := 0
	globalAggregateDelayedOffsetCount := 0
	for _, config := range loadedConfigs {
		if slices.Contains(checker.SymbolsToBeDelisted, config.Name) {
			continue
		}

		log.Debug().Int32("id", config.ID).Str("name", config.Name).Msg("checking config offset")
		localAggregateOffsetResult, err := db.QueryRowTransient[OffsetResultEach](ctx, serviceDB, fmt.Sprintf(GetSingleLocalAggregateOffset, config.ID), nil)
		if err != nil {
			log.Error().Err(err).Msg("Error getting local aggregate offset")
			return
		}
		if localAggregateOffsetResult.Delay > DelayThreshold.Seconds() {
			localAggregateAlarmCount[config.ID]++
			if localAggregateAlarmCount[config.ID] > AlarmOffsetPerPair {
				msg += fmt.Sprintf("(local aggregate offset delayed) %s: %v seconds\n", config.Name, localAggregateOffsetResult.Delay)
				localAggregateAlarmCount[config.ID] = 0
			} else if localAggregateAlarmCount[config.ID] > AlarmOffsetInTotal {
				localAggregateDelayedOffsetCount++
			}
		} else {
			localAggregateAlarmCount[config.ID] = 0
		}

		globalAggregateOffsetResult, err := db.QueryRowTransient[OffsetResultEach](ctx, serviceDB, fmt.Sprintf(GetSingleGlobalAggregateOffset, config.ID), nil)
		if err != nil {
			log.Error().Err(err).Msg("Error getting global aggregate offset")
			return
		}
		if globalAggregateOffsetResult.Delay > DelayThreshold.Seconds() {
			globalAggregateAlarmCount[config.ID]++
			if globalAggregateAlarmCount[config.ID] > AlarmOffsetPerPair {
				msg += fmt.Sprintf("(global aggregate offset delayed) %s: %v seconds\n", config.Name, globalAggregateOffsetResult.Delay)
				globalAggregateAlarmCount[config.ID] = 0
			} else if globalAggregateAlarmCount[config.ID] > AlarmOffsetInTotal {
				globalAggregateDelayedOffsetCount++
			}
		} else {
			globalAggregateAlarmCount[config.ID] = 0
		}

		latestAggregateResult, err := db.QueryRowTransient[LatestAggregateEach](ctx, serviceDB, fmt.Sprintf(GetSingleLatestAggregates, config.ID, config.ID), nil)
		if err != nil {
			log.Error().Err(err).Msg("Error getting latest aggregate")
			return
		}

		if math.Abs(latestAggregateResult.LocalAggregate-latestAggregateResult.GlobalAggregate)/latestAggregateResult.LocalAggregate > PriceDifferenceThreshold {
			msg += fmt.Sprintf("(latest aggregate price difference) %s: %v (globalAggregate: %v, localAggregate: %v)\n", config.Name, latestAggregateResult.LocalAggregate-latestAggregateResult.GlobalAggregate, latestAggregateResult.GlobalAggregate, latestAggregateResult.LocalAggregate)
		}

		time.Sleep(100 * time.Millisecond) // sleep to reduce pgsql stress
	}

	if localAggregateDelayedOffsetCount > 0 {
		msg += fmt.Sprintf("%d local aggregate offsets delayed\n", localAggregateDelayedOffsetCount)
	}

	if globalAggregateDelayedOffsetCount > 0 {
		msg += fmt.Sprintf("%d global aggregate offsets delayed\n", globalAggregateDelayedOffsetCount)
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}
}
