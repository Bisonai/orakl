package common

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// HeartbeatPoll periodically calls fetchPrice and emits the result as a
// FeedData on every successful read.  The loop runs in the foreground —
// callers must invoke it from the goroutine that owns the polling lifetime.
// Errors from fetchPrice are logged and skipped (the loop keeps ticking).
//
// This is the testable extraction of the heartbeat pattern shared by the
// UniswapV3-style DEX fetchers (uniswap, pancakeswap, capybara).  Putting it
// in one place ensures all three providers behave identically and makes the
// "polling continues across the lifetime of ctx" invariant unit-testable
// without needing to mock a chain client.
func HeartbeatPoll(
	ctx context.Context,
	interval time.Duration,
	player string,
	feedID int32,
	feedName string,
	fetchPrice func(context.Context) (*float64, error),
	emit func(*FeedData),
) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			price, err := fetchPrice(ctx)
			if err != nil {
				log.Error().Str("Player", player).Err(err).Msg("failed to poll slot0()")
				continue
			}
			if price == nil {
				continue
			}

			// TODO(diag): drop after IDRX-USDT polling confirmed.
			log.Info().Str("Player", player).Int32("feedID", feedID).Str("name", feedName).Float64("price", *price).Msg("DIAG polled price")

			now := time.Now()
			emit(&FeedData{
				FeedID:    feedID,
				Value:     *price,
				Timestamp: &now,
			})
		case <-ctx.Done():
			return
		}
	}
}
