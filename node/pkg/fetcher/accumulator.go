package fetcher

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
)

func NewAccumulator(interval time.Duration) *Accumulator {
	return &Accumulator{
		Interval: interval,
	}
}

func (a *Accumulator) Run(ctx context.Context) {
	accumulatorCtx, cancel := context.WithCancel(ctx)
	a.accumulatorCtx = accumulatorCtx
	a.cancel = cancel
	a.isRunning = true

	ticker := time.NewTicker(DefaultLocalAggregateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("Player", "Fetcher").Msg("fetcher local aggregates channel goroutine stopped")
			return
		case <-ticker.C:
			go a.accumulatorJob(ctx)
		}
	}
}

func (a *Accumulator) accumulatorJob(ctx context.Context) {
	if len(a.accumulatorChannel) == 0 {
		return
	}
	localAggregatesDataRedis := make(map[string]interface{})
	var localAggregatesDataPgsql [][]any

loop:
	for {
		select {
		case data := <-a.accumulatorChannel:
			localAggregatesDataRedis[keys.LocalAggregateKey(data.ConfigID)] = data
			localAggregatesDataPgsql = append(localAggregatesDataPgsql, []any{data.ConfigID, int64(data.Value), data.Timestamp})
		default:
			break loop
		}
	}

	redisErr := db.MSetObject(ctx, localAggregatesDataRedis)
	_, pgsqlErr := db.BulkCopy(ctx, "local_aggregates", []string{"config_id", "value", "timestamp"}, localAggregatesDataPgsql)

	if redisErr != nil || pgsqlErr != nil {
		log.Error().Err(redisErr).Err(pgsqlErr).Msg("failed to save local aggregates")
	}
}
