// Package history keeps a low-resolution price history per symbol (the last
// value of each fixed time window) in a local SQLite file.
package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

const (
	EnvEnabled   = "DAL_HISTORY_ENABLED"
	EnvPath      = "DAL_HISTORY_PATH"
	EnvInterval  = "DAL_HISTORY_INTERVAL"
	EnvRetention = "DAL_HISTORY_RETENTION"

	DefaultInterval  = 5 * time.Minute
	DefaultRetention = 168 * time.Hour
	// FileName is the history file created next to DAL_SQLITE_PATH.
	FileName = "history.db"

	PruneInterval = time.Hour
	closeTimeout  = 5 * time.Second
)

// FlushInterval is how often buffered samples are written. A var so tests can shorten it.
var FlushInterval = 10 * time.Second

const schema = `
CREATE TABLE IF NOT EXISTS history (
	symbol         TEXT    NOT NULL,
	bucket_start   INTEGER NOT NULL, -- unix ms
	value          INTEGER NOT NULL,
	decimals       TEXT    NOT NULL,
	aggregate_time INTEGER NOT NULL, -- unix ms
	PRIMARY KEY (symbol, bucket_start)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS history_bucket_start ON history (bucket_start);`

// The newer aggregate wins within a window, also across processes sharing the file.
const upsertQuery = `INSERT INTO history (symbol, bucket_start, value, decimals, aggregate_time) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (symbol, bucket_start) DO UPDATE SET
	value = excluded.value, decimals = excluded.decimals, aggregate_time = excluded.aggregate_time
WHERE excluded.aggregate_time >= history.aggregate_time`

type Config struct {
	Path      string
	Interval  time.Duration
	Retention time.Duration
}

// ConfigFromEnv reads the history settings and reports whether history is
// enabled: on by default, off with DAL_HISTORY_ENABLED=false or when neither
// DAL_HISTORY_PATH nor DAL_SQLITE_PATH is set.
func ConfigFromEnv() (Config, bool, error) {
	if v := os.Getenv(EnvEnabled); v != "" {
		on, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, false, fmt.Errorf("%s: %w", EnvEnabled, err)
		}
		if !on {
			return Config{}, false, nil
		}
	}

	cfg := Config{Path: os.Getenv(EnvPath), Interval: DefaultInterval, Retention: DefaultRetention}
	if cfg.Path == "" {
		keysPath := os.Getenv(keystore.EnvPath)
		if keysPath == "" {
			return Config{}, false, nil
		}
		cfg.Path = filepath.Join(filepath.Dir(keysPath), FileName)
	}

	for _, d := range []struct {
		env string
		dst *time.Duration
	}{{EnvInterval, &cfg.Interval}, {EnvRetention, &cfg.Retention}} {
		v := os.Getenv(d.env)
		if v == "" {
			continue
		}
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, false, fmt.Errorf("%s: %w", d.env, err)
		}
		*d.dst = parsed
	}
	if err := cfg.validate(); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

func (c Config) validate() error {
	if c.Interval < time.Second || c.Interval%time.Millisecond != 0 {
		return fmt.Errorf("%s must be a whole number of ms and at least 1s, got %s", EnvInterval, c.Interval)
	}
	if c.Retention < c.Interval {
		return fmt.Errorf("%s (%s) must not be shorter than %s (%s)", EnvRetention, c.Retention, EnvInterval, c.Interval)
	}
	return nil
}

// Point is one sample: the bucket start (unix ms) and the raw value.
type Point struct {
	Time  int64  `json:"time"`
	Value string `json:"value"`
}

// Series is the history of one symbol, time-ordered.
type Series struct {
	Decimals string  `json:"decimals"`
	Points   []Point `json:"points"`
}

type bucketKey struct {
	symbol string
	bucket int64
}

type sample struct {
	value         int64
	decimals      string
	aggregateTime int64
}

type Store struct {
	db  *sql.DB
	cfg Config

	mu      sync.Mutex
	pending map[bucketKey]sample

	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// Open opens (creating if missing) the history file and ensures the schema.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	// SQLite decodes '%HH' in file: URIs, so '%' would open a different file.
	if strings.ContainsAny(cfg.Path, "?#%") {
		return nil, fmt.Errorf("history path %q must not contain '?', '#' or '%%'", cfg.Path)
	}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	// take the write lock at BEGIN, so a concurrent writer waits (busy_timeout) instead of failing
	q.Add("_txlock", "immediate")
	dsn := "file:" + cfg.Path + "?" + q.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", cfg.Path, err)
	}
	if _, err = db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite schema %s: %w", cfg.Path, err)
	}
	return &Store{db: db, cfg: cfg, pending: make(map[bucketKey]sample)}, nil
}

func (s *Store) Interval() time.Duration  { return s.cfg.Interval }
func (s *Store) Retention() time.Duration { return s.cfg.Retention }

func (s *Store) bucketStart(ms int64) int64 {
	return floorTo(ms, s.cfg.Interval.Milliseconds())
}

func floorTo(ms, step int64) int64 {
	r := ms % step
	if r < 0 {
		r += step
	}
	return ms - r
}

// Record buffers value as the latest sample of its window (by aggregate time
// ts); the newest sample per window wins. It never touches the database.
func (s *Store) Record(symbol string, value int64, decimals string, ts time.Time) {
	ms := ts.UnixMilli()
	key := bucketKey{symbol: symbol, bucket: s.bucketStart(ms)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.pending[key]; ok && old.aggregateTime > ms {
		return
	}
	s.pending[key] = sample{value: value, decimals: decimals, aggregateTime: ms}
}

// Flush writes the buffered samples in one transaction. On failure they are
// kept (unless superseded) for the next flush.
func (s *Store) Flush(ctx context.Context) error {
	s.mu.Lock()
	batch := s.pending
	s.pending = make(map[bucketKey]sample, len(batch))
	s.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}

	if err := s.write(ctx, batch); err != nil {
		// keep for retry, but never past retention, so a persistent failure does not grow the buffer forever
		expired := s.bucketStart(time.Now().Add(-s.cfg.Retention).UnixMilli())
		s.mu.Lock()
		for key, smp := range batch {
			if key.bucket < expired {
				continue
			}
			if cur, ok := s.pending[key]; !ok || cur.aggregateTime < smp.aggregateTime {
				s.pending[key] = smp
			}
		}
		s.mu.Unlock()
		return err
	}
	return nil
}

func (s *Store) write(ctx context.Context, batch map[bucketKey]sample) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, upsertQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for key, smp := range batch {
		if _, err := stmt.ExecContext(ctx, key.symbol, key.bucket, smp.value, smp.decimals, smp.aggregateTime); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Prune deletes windows that started before now - retention.
func (s *Store) Prune(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM history WHERE bucket_start < ?", now.Add(-s.cfg.Retention).UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Start runs the flush and prune loops until ctx is done or Close is called.
func (s *Store) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.prune(ctx)
		flush := time.NewTicker(FlushInterval)
		defer flush.Stop()
		prune := time.NewTicker(PruneInterval)
		defer prune.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-flush.C:
				if err := s.Flush(ctx); err != nil && ctx.Err() == nil {
					log.Error().Err(err).Str("Player", "DalHistory").Msg("failed to flush history")
				}
			case <-prune.C:
				s.prune(ctx)
			}
		}
	}()
}

func (s *Store) prune(ctx context.Context) {
	n, err := s.Prune(ctx, time.Now())
	if err != nil {
		if ctx.Err() == nil {
			log.Error().Err(err).Str("Player", "DalHistory").Msg("failed to prune history")
		}
		return
	}
	log.Debug().Str("Player", "DalHistory").Int64("rows", n).Msg("pruned history")
}

// Close stops the loops, flushes the buffered samples and closes the file.
func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
			<-s.done
		}
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		flushErr := s.Flush(ctx)
		s.closeErr = errors.Join(flushErr, s.db.Close())
	})
	return s.closeErr
}

// Query returns the history of symbols from windows starting at or after
// since (rounded up to a step boundary), downsampled to step (a multiple of the interval) by keeping the last
// sample of each step; points are labelled with the step start. Every
// requested symbol is in the result, with no points if it has no history.
func (s *Store) Query(ctx context.Context, symbols []string, since time.Time, step time.Duration) (map[string]Series, error) {
	stepMs := step.Milliseconds()
	if stepMs <= 0 {
		return nil, fmt.Errorf("invalid step %s", step)
	}
	stmt, err := s.db.PrepareContext(ctx, "SELECT bucket_start, value, decimals FROM history WHERE symbol = ? AND bucket_start >= ? ORDER BY bucket_start")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// start at a step boundary, so the first point covers a whole step
	sinceMs := floorTo(since.UnixMilli()+stepMs-1, stepMs)
	out := make(map[string]Series, len(symbols))
	for _, symbol := range symbols {
		series, err := querySymbol(ctx, stmt, symbol, sinceMs, stepMs)
		if err != nil {
			return nil, err
		}
		out[symbol] = series
	}
	return out, nil
}

func querySymbol(ctx context.Context, stmt *sql.Stmt, symbol string, sinceMs, stepMs int64) (Series, error) {
	rows, err := stmt.QueryContext(ctx, symbol, sinceMs)
	if err != nil {
		return Series{}, err
	}
	defer rows.Close()

	series := Series{Points: []Point{}}
	for rows.Next() {
		var bucket, value int64
		var decimals string
		if err := rows.Scan(&bucket, &value, &decimals); err != nil {
			return Series{}, err
		}
		// a decimals change (config reload) rescales values: keep only points at the latest decimals
		if decimals != series.Decimals {
			series.Points = series.Points[:0]
		}
		p := Point{Time: floorTo(bucket, stepMs), Value: strconv.FormatInt(value, 10)}
		if n := len(series.Points); n > 0 && series.Points[n-1].Time == p.Time {
			series.Points[n-1] = p
		} else {
			series.Points = append(series.Points, p)
		}
		series.Decimals = decimals
	}
	return series, rows.Err()
}
