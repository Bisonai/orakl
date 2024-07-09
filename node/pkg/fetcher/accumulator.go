package fetcher

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	goroutine_pool "bisonai.com/orakl/node/pkg/utils/goroutine-pool"
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

	pool := goroutine_pool.NewPool()
	pool.Run(accumulatorCtx)

	ticker := time.NewTicker(a.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pool.AddJob(func() {
				a.accumulatorJob(accumulatorCtx)
			})
		case <-ctx.Done():
			log.Debug().Str("Player", "Fetcher").Msg("fetcher local aggregates channel goroutine stopped")
			return
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
			localAggregatesDataPgsql = append(localAggregatesDataPgsql, []any{data.ConfigID, int64(data.Value), data.Timestamp})

			redisKey := keys.LocalAggregateKey(data.ConfigID)
			existingData, exists := localAggregatesDataRedis[redisKey]
			if exists && existingData.(LocalAggregate).Timestamp.After(data.Timestamp) {
				continue
			}
			localAggregatesDataRedis[redisKey] = data
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
