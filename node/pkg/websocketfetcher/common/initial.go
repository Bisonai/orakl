package common

import (
	"context"
	"errors"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/rs/zerolog/log"
)

// InitialPriceBackoff controls the retry delay of FetchInitialPrice.  The
// delay starts at Initial and doubles after every failed attempt, capped
// at Max.  AttemptTimeout (if > 0) bounds each fetchPrice call so a hung
// read on a half-open connection cannot stall the retry loop.
type InitialPriceBackoff struct {
	Initial        time.Duration
	Max            time.Duration
	AttemptTimeout time.Duration
}

var errNilInitialPrice = errors.New("fetchPrice returned nil price")

// minInitialPriceBackoff guards against a zero/negative backoff turning
// the retry loop into a busy loop.
const minInitialPriceBackoff = 10 * time.Millisecond

// DefaultInitialPriceBackoff is used by the DEX providers: 1s, 2s, 4s, ...
// capped at 60s — roughly the heartbeat poll cadence, so a pool that keeps
// failing logs no more often than HeartbeatPoll would.
var DefaultInitialPriceBackoff = InitialPriceBackoff{
	Initial:        1 * time.Second,
	Max:            60 * time.Second,
	AttemptTimeout: 30 * time.Second,
}

// IsPermanentInitialPriceError reports whether err can never succeed on
// retry: an invalid feed definition (providers wrap definition unmarshal /
// poolId errors in ErrFetcherInvalidDexFetcherDefinition), an empty pool
// address, or a chainId this node has no websocket URL (or V4 deployment)
// for.  Everything else (dial / reset / closed conn / timeout / RPC,
// including RPC response decode errors / result shape) is transient.
func IsPermanentInitialPriceError(err error) bool {
	return errors.Is(err, errorSentinel.ErrFetcherNoMatchingChainID) ||
		errors.Is(err, errorSentinel.ErrFetcherInvalidDexFetcherDefinition) ||
		errors.Is(err, errorSentinel.ErrChainEmptyAddressParam)
}

// FetchInitialPrice calls fetchPrice until it returns a price, retrying
// transient errors with capped exponential backoff.  It returns early
// with the error on a permanent error (see IsPermanentInitialPriceError)
// or with ctx.Err() once ctx is done.
//
// Previously the DEX providers returned from run() on the first failed
// initial read, leaving the pool with no subscription and no heartbeat
// poll until the process restarted (#2609).
func FetchInitialPrice(
	ctx context.Context,
	backoff InitialPriceBackoff,
	player string,
	feedID int32,
	feedName string,
	fetchPrice func(context.Context) (*float64, error),
) (*float64, error) {
	maxDelay := max(backoff.Max, minInitialPriceBackoff)
	delay := min(max(backoff.Initial, minInitialPriceBackoff), maxDelay)
	for attempt := 1; ; attempt++ {
		price, err := fetchAttempt(ctx, backoff.AttemptTimeout, fetchPrice)
		if err == nil && price != nil {
			if attempt > 1 {
				log.Info().Str("Player", player).Int32("feedID", feedID).Str("name", feedName).
					Int("attempt", attempt).Msg("initial price fetched after retry")
			}
			return price, nil
		}
		if err == nil {
			err = errNilInitialPrice
		}
		if IsPermanentInitialPriceError(err) {
			return nil, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		log.Warn().Str("Player", player).Int32("feedID", feedID).Str("name", feedName).
			Int("attempt", attempt).Dur("retryIn", delay).Err(err).
			Msg("failed to get initial price, retrying")

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}

		delay = min(delay*2, maxDelay)
	}
}

func fetchAttempt(
	ctx context.Context,
	timeout time.Duration,
	fetchPrice func(context.Context) (*float64, error),
) (*float64, error) {
	if timeout <= 0 {
		return fetchPrice(ctx)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return fetchPrice(attemptCtx)
}
