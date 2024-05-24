package fetcher

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func NewStreamer(interval time.Duration) *Streamer {
	return &Streamer{
		Interval: interval,
	}
}

func (s *Streamer) Run(ctx context.Context) {
	streamerCtx, cancel := context.WithCancel(ctx)
	s.streamerCtx = streamerCtx
	s.cancel = cancel
	s.isRunning = true

	ticker := time.NewTicker(s.Interval)
	go func() {
		for {
			select {
			case <-s.streamerCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := s.Job(s.streamerCtx)
				if err != nil {
					log.Error().Str("Player", "Streamer").Err(err).Msg("error in streamerJob")
				}
			}
		}
	}()
}

func (s *Streamer) Job(ctx context.Context) error {
	log.Debug().Str("Player", "Streamer").Msg("streamerJob")
	result, err := getFeedDataBuffer(ctx)
	if err != nil {
		log.Error().Str("Player", "Streamer").Err(err).Msg("error in getFeedDataBuffer")
		return err
	}
	return copyFeedData(ctx, result)
}
