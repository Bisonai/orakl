package aggregator

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// roundFlushInterval is the minimum spacing between round writes. A crash
// loses at most the rounds advanced since the last write (< 1 round per
// config at the 400ms aggregate interval, while writes succeed); Run starts
// at stored+1 and LeaderJob increments before publishing, so a restarted
// leader's first round is stored+2, which covers that one lost round.
const roundFlushInterval = 100 * time.Millisecond

// roundFlushRetryDelay is the wait before retrying a failed round write.
const roundFlushRetryDelay = time.Second

// roundStore persists the latest round per config name (nodestate.Store).
type roundStore interface {
	LatestRound(ctx context.Context, configName string) (int32, bool, error)
	SaveRounds(ctx context.Context, rounds map[string]int32) error
}

// roundPersister persists rounds off the round path. Advance only records the
// value in memory and wakes a single writer goroutine, which coalesces all
// pending values into one write per flush.
type roundPersister struct {
	store roundStore

	mu      sync.Mutex
	known   map[string]int32 // highest round seen per config (pending or written)
	pending map[string]int32 // not yet written

	flushMu sync.Mutex // serializes writes (run loop vs Flush on shutdown)
	wake    chan struct{}
}

func newRoundPersister(store roundStore) *roundPersister {
	return &roundPersister{
		store:   store,
		known:   map[string]int32{},
		pending: map[string]int32{},
		wake:    make(chan struct{}, 1),
	}
}

// Advance records round for name if it is higher than any seen; never blocks
// on I/O.
func (p *roundPersister) Advance(name string, round int32) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if round <= p.known[name] {
		p.mu.Unlock()
		return
	}
	p.known[name] = round
	p.pending[name] = round
	p.mu.Unlock()

	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Latest returns the highest known round for name: the in-memory value if
// any, else the stored one (0 if none).
func (p *roundPersister) Latest(ctx context.Context, name string) (int32, error) {
	p.mu.Lock()
	round, ok := p.known[name]
	p.mu.Unlock()
	if ok {
		return round, nil
	}

	stored, found, err := p.store.LatestRound(ctx, name)
	if err != nil || !found {
		return 0, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if stored > p.known[name] {
		p.known[name] = stored
	}
	return p.known[name], nil
}

// Flush writes all pending rounds. On error they are put back (keeping the
// higher value) for the next flush.
func (p *roundPersister) Flush(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.flushMu.Lock()
	defer p.flushMu.Unlock()

	p.mu.Lock()
	batch := p.pending
	p.pending = map[string]int32{}
	p.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}

	err := p.store.SaveRounds(ctx, batch)
	if err != nil {
		p.mu.Lock()
		for name, round := range batch {
			if round > p.pending[name] {
				p.pending[name] = round
			}
		}
		p.mu.Unlock()
	}
	return err
}

// run flushes on every wake, at most once per roundFlushInterval, until ctx
// is done; then flushes what is left.
func (p *roundPersister) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := p.Flush(flushCtx); err != nil {
				log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to persist rounds on shutdown")
			}
			cancel()
			return
		case <-p.wake:
		}

		delay := roundFlushInterval
		if err := p.Flush(ctx); err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to persist rounds, will retry")
			delay = roundFlushRetryDelay
			select {
			case p.wake <- struct{}{}:
			default:
			}
		}

		select {
		case <-ctx.Done():
		case <-time.After(delay):
		}
	}
}
