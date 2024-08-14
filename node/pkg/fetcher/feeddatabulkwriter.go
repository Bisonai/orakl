package fetcher

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func NewFeedDataBulkWriter(interval time.Duration, feedDataDumpChannel chan *FeedData) *FeedDataBulkWriter {
	return &FeedDataBulkWriter{
		Interval:            interval,
		FeedDataDumpChannel: feedDataDumpChannel,
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
	select {
	case <-ctx.Done():
		return nil
	case entry := <-s.FeedDataDumpChannel:
		result := []*FeedData{entry}
		for entry := range s.FeedDataDumpChannel {
			result = append(result, entry)
		}
		return copyFeedData(ctx, result)
	default:
		return nil
	}

}
