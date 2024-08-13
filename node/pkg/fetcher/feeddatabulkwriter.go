package fetcher

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func NewFeedDataBulkWriter(interval time.Duration) *FeedDataBulkWriter {
	return &FeedDataBulkWriter{
		Interval: interval,
	}
}

func (s *FeedDataBulkWriter) Run(ctx context.Context) {
	writerCtx, cancel := context.WithCancel(ctx)
	s.writerCtx = writerCtx
	s.cancel = cancel
	s.isRunning = true

	ticker := time.NewTicker(s.Interval)
	go func() {
		for {
			select {
			case <-s.writerCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := s.Job(s.writerCtx)
				if err != nil {
					log.Error().Str("Player", "FeedDataBulkWriter").Err(err).Msg("error in FeedDataBulkWriterJob")
				}
			}
		}
	}()
}

func (s *FeedDataBulkWriter) Job(ctx context.Context) error {
	log.Debug().Str("Player", "FeedDataBulkWriter").Msg("FeedDataBulkWriterJob")
	result, err := getFeedDataBuffer(ctx)
	if err != nil {
		log.Error().Str("Player", "FeedDataBulkWriter").Err(err).Msg("error in getFeedDataBuffer")
		return err
	}
	return copyFeedData(ctx, result)
}
