package fetcher

import (
	"context"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"github.com/rs/zerolog/log"
)

func NewLocalAggregateBulkWriter(interval time.Duration) *LocalAggregateBulkWriter {
	return &LocalAggregateBulkWriter{
		Interval: interval,
	}
}

func (a *LocalAggregateBulkWriter) Run(ctx context.Context) {
	bulkWriterCtx, cancel := context.WithCancel(ctx)
	a.bulkWriterCtx = bulkWriterCtx
	a.cancel = cancel
	a.isRunning = true
	done := make(chan struct{})
	a.done = done

	ticker := time.NewTicker(a.Interval)
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				go a.bulkWriterJob(bulkWriterCtx)
			case <-bulkWriterCtx.Done():
				log.Debug().Str("Player", "Fetcher").Msg("fetcher local aggregates channel goroutine stopped")
				return
			}
		}
	}()
}

func (a *LocalAggregateBulkWriter) bulkWriterJob(ctx context.Context) {
	// a tick can race with stop; don't drain into a BulkCopy that will fail
	if ctx.Err() != nil || len(a.localAggregatesChannel) == 0 {
		return
	}
	var localAggregatesDataPgsql [][]any

loop:
	for {
		select {
		case data := <-a.localAggregatesChannel:
			localAggregatesDataPgsql = append(localAggregatesDataPgsql, []any{data.ConfigID, data.Value, data.Timestamp})
		default:
			break loop
		}
	}

	_, pgsqlErr := db.BulkCopy(ctx, "local_aggregates", []string{"config_id", "value", "timestamp"}, localAggregatesDataPgsql)

	if pgsqlErr != nil {
		log.Error().Err(pgsqlErr).Msg("failed to save local aggregates")
	}
}
