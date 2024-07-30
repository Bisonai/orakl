package fetcher

import (
	"context"
	"time"

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

	ticker := time.NewTicker(a.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go a.accumulatorJob(accumulatorCtx)
		case <-ctx.Done():
			log.Debug().Str("Player", "Fetcher").Msg("fetcher local aggregates channel goroutine stopped")
			return
		}
	}
}

func (a *Accumulator) accumulatorJob(ctx context.Context) {
	start := time.Now()
	if len(a.accumulatorChannel) == 0 {
		return
	}
	var localAggregatesDataPgsql [][]any

loop:
	for {
		select {
		case data := <-a.accumulatorChannel:
			localAggregatesDataPgsql = append(localAggregatesDataPgsql, []any{data.ConfigID, int64(data.Value), data.Timestamp})
		default:
			break loop
		}
	}

	_, pgsqlErr := db.BulkCopy(ctx, "local_aggregates", []string{"config_id", "value", "timestamp"}, localAggregatesDataPgsql)

	if pgsqlErr != nil {
		log.Error().Err(pgsqlErr).Msg("failed to save local aggregates")
	}

	diff := time.Since(start).Milliseconds()
	if diff > 200 {
		log.Warn().Str("Player", "Fetcher").Msgf("accumulatorJob finished in %d ms", diff)
	}
}
