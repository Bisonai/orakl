package helper

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// fallbackWarnInterval is the minimum gap between delegator-fallback warnings
// per reason and purpose.
const fallbackWarnInterval = time.Minute

// fallbackWarner is process-wide: rotation builds a new ChainHelper per call.
var fallbackWarner = newRateLimitedWarner(fallbackWarnInterval, time.Now)

// rateLimitedWarner logs at most one warning per key per interval, with the
// number of warnings suppressed since the previous one.
type rateLimitedWarner struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	last     map[string]time.Time
	skipped  map[string]int
}

func newRateLimitedWarner(interval time.Duration, now func() time.Time) *rateLimitedWarner {
	return &rateLimitedWarner{
		interval: interval,
		now:      now,
		last:     map[string]time.Time{},
		skipped:  map[string]int{},
	}
}

// warn logs a delegator error that made a tx fall back to direct (paid by
// the sending key). err must not carry the token; net/http errors do not.
func (w *rateLimitedWarner) warn(reason, purpose string, err error) {
	key := reason + "|" + purpose
	w.mu.Lock()
	now := w.now()
	if last, ok := w.last[key]; ok && now.Sub(last) < w.interval {
		w.skipped[key]++
		w.mu.Unlock()
		return
	}
	suppressed := w.skipped[key]
	w.last[key] = now
	w.skipped[key] = 0
	w.mu.Unlock()

	log.Warn().
		Err(err).
		Str("reason", reason).
		Str("purpose", purpose).
		Int("suppressed", suppressed).
		Msg("delegator sign failed, falling back to direct tx paid by the sending key")
}
