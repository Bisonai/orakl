// Package configreload re-reads an orakl-config bundle (see configfetch) while
// a service runs and hands changed content to the service's apply function.
//
// A reload runs every CONFIG_RELOAD_INTERVAL and on SIGHUP. It is skipped when
// the fetched bytes equal the last applied ones, and refused (current config
// kept) when the fetch or parse fails or the bundle looks wrong: empty, an
// entry without name, a duplicate name, or more than MaxDropRatio of the
// entries gone at once. A refused or failed reload is retried on the next poll.
//
// Environment:
//   - CONFIG_RELOAD_INTERVAL: Go duration (e.g. "5m"), default DefaultInterval;
//     "0" disables polling (SIGHUP still reloads).
package configreload

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bisonai.com/miko/node/pkg/utils/configfetch"
	"github.com/rs/zerolog/log"
)

const (
	IntervalEnv     = "CONFIG_RELOAD_INTERVAL"
	DefaultInterval = 5 * time.Minute
	// MaxDropRatio is the largest share of entries one reload may remove.
	// Bigger drops are refused as a likely broken bundle; restart to apply.
	MaxDropRatio = 0.5
)

// Interval returns the poll interval from CONFIG_RELOAD_INTERVAL; 0 = disabled.
// An invalid or negative value falls back to DefaultInterval.
func Interval() time.Duration {
	v := os.Getenv(IntervalEnv)
	if v == "" {
		return DefaultInterval
	}
	if v == "0" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		log.Warn().Str("value", v).Dur("default", DefaultInterval).Msg("invalid " + IntervalEnv + ", using default")
		return DefaultInterval
	}
	return d
}

// NotifyHUP returns a channel receiving SIGHUP (which then no longer
// terminates the process).
func NotifyHUP() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	return ch
}

// Reloader reloads a bundle of []T. Reloads are serialized.
type Reloader[T any] struct {
	path  string
	name  func(T) string
	apply func(context.Context, []T) error

	mu       sync.Mutex
	count    int // entries of the applied bundle
	lastHash [sha256.Size]byte
	hasHash  bool
}

// New returns a reloader for path. count is the number of entries the service
// loaded at startup (the reference for the drop guard); name returns an
// entry's unique name; apply applies a validated bundle and must leave the
// service consistent on error (the bundle is retried on the next reload).
func New[T any](path string, count int, name func(T) string, apply func(context.Context, []T) error) *Reloader[T] {
	return &Reloader[T]{path: path, count: count, name: name, apply: apply}
}

// Reload fetches the bundle once and applies it if its content changed. It
// reports whether apply was called and succeeded.
func (r *Reloader[T]) Reload(ctx context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	body, err := configfetch.Fetch(ctx, r.path)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256(body)
	if r.hasHash && hash == r.lastHash {
		return false, nil
	}

	var entries []T
	if err := json.Unmarshal(body, &entries); err != nil {
		return false, fmt.Errorf("decode config %s: %w", r.path, err)
	}
	if err := r.check(entries); err != nil {
		return false, err
	}
	if err := r.apply(ctx, entries); err != nil {
		return false, fmt.Errorf("apply config %s: %w", r.path, err)
	}
	r.lastHash, r.hasHash, r.count = hash, true, len(entries)
	return true, nil
}

func (r *Reloader[T]) check(entries []T) error {
	if len(entries) == 0 {
		return fmt.Errorf("refusing config %s: bundle is empty", r.path)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		n := r.name(e)
		if n == "" {
			return fmt.Errorf("refusing config %s: entry without name", r.path)
		}
		if _, dup := seen[n]; dup {
			return fmt.Errorf("refusing config %s: duplicate entry %s", r.path, n)
		}
		seen[n] = struct{}{}
	}
	if float64(len(entries)) < float64(r.count)*(1-MaxDropRatio) {
		return fmt.Errorf("refusing config %s: %d entries, down from %d (max drop %.0f%%; restart to apply)", r.path, len(entries), r.count, MaxDropRatio*100)
	}
	return nil
}

// Run reloads every interval (never if interval <= 0) and on every value from
// trigger (may be nil) until ctx is done. Errors are logged.
func (r *Reloader[T]) Run(ctx context.Context, interval time.Duration, trigger <-chan os.Signal) {
	var tick <-chan time.Time
	if interval > 0 {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		tick = ticker.C
	}
	log.Info().Str("path", r.path).Dur("interval", interval).Msg("config reload enabled (also on SIGHUP)")
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
		case <-trigger:
			log.Info().Str("path", r.path).Msg("config reload triggered")
		}
		applied, err := r.Reload(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			log.Error().Err(err).Str("path", r.path).Msg("config reload failed, keeping current config")
		case applied:
			log.Info().Str("path", r.path).Msg("config reloaded")
		}
	}
}
