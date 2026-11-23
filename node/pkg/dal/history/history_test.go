package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var t0 = time.UnixMilli(1_700_000_100_000) // inside a 5m window, not at its start

func testConfig(t *testing.T) Config {
	return Config{Path: filepath.Join(t.TempDir(), FileName), Interval: 5 * time.Minute, Retention: 168 * time.Hour}
}

func openStore(t *testing.T, cfg Config) *Store {
	t.Helper()
	s, err := Open(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func query(t *testing.T, s *Store, step time.Duration, symbols ...string) map[string]Series {
	t.Helper()
	out, err := s.Query(context.Background(), symbols, t0.Add(-24*time.Hour), step)
	require.NoError(t, err)
	return out
}

func window(ts time.Time) int64 {
	return floorTo(ts.UnixMilli(), (5 * time.Minute).Milliseconds())
}

func TestLastValueWinsWithinWindow(t *testing.T) {
	ctx := context.Background()
	s := openStore(t, testConfig(t))

	s.Record("BTC-USDT", 1, "8", t0)
	s.Record("BTC-USDT", 3, "8", t0.Add(20*time.Second))
	s.Record("BTC-USDT", 2, "8", t0.Add(10*time.Second)) // older arrives late: ignored
	require.NoError(t, s.Flush(ctx))
	// a later flush in the same window updates the row
	s.Record("BTC-USDT", 4, "8", t0.Add(30*time.Second))
	require.NoError(t, s.Flush(ctx))
	// next window gets its own row
	s.Record("BTC-USDT", 5, "8", t0.Add(5*time.Minute))
	require.NoError(t, s.Flush(ctx))

	got := query(t, s, 5*time.Minute, "BTC-USDT")["BTC-USDT"]
	assert.Equal(t, "8", got.Decimals)
	assert.Equal(t, []Point{
		{Time: window(t0), Value: "4"},
		{Time: window(t0.Add(5 * time.Minute)), Value: "5"},
	}, got.Points)
}

func TestUpsertIdempotentAcrossWriters(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	a := openStore(t, cfg)
	b := openStore(t, cfg) // second process on the same file (overlapping deploy)

	a.Record("ETH-USDT", 10, "8", t0.Add(20*time.Second))
	b.Record("ETH-USDT", 9, "8", t0.Add(10*time.Second))
	require.NoError(t, a.Flush(ctx))
	require.NoError(t, b.Flush(ctx)) // older value from the other writer must not win
	a.Record("ETH-USDT", 10, "8", t0.Add(20*time.Second))
	require.NoError(t, a.Flush(ctx)) // same value again: no change

	for _, s := range []*Store{a, b} {
		got := query(t, s, 5*time.Minute, "ETH-USDT")["ETH-USDT"]
		assert.Equal(t, []Point{{Time: window(t0), Value: "10"}}, got.Points)
	}
	var n int
	require.NoError(t, a.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&n))
	assert.Equal(t, 1, n)
}

func TestPrunePastRetention(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	cfg.Retention = time.Hour
	s := openStore(t, cfg)

	s.Record("BTC-USDT", 1, "8", t0.Add(-2*time.Hour))
	s.Record("BTC-USDT", 2, "8", t0.Add(-30*time.Minute))
	s.Record("BTC-USDT", 3, "8", t0)
	require.NoError(t, s.Flush(ctx))

	n, err := s.Prune(ctx, t0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	got := query(t, s, 5*time.Minute, "BTC-USDT")["BTC-USDT"]
	assert.Equal(t, []string{"2", "3"}, values(got.Points))
}

func TestDownsampleKeepsLastPerStep(t *testing.T) {
	ctx := context.Background()
	s := openStore(t, testConfig(t))
	hour := floorTo(t0.UnixMilli(), time.Hour.Milliseconds())
	base := time.UnixMilli(hour)
	for i := 0; i < 24; i++ { // 2h of 5m windows
		s.Record("BTC-USDT", int64(i), "8", base.Add(time.Duration(i)*5*time.Minute))
	}
	require.NoError(t, s.Flush(ctx))

	got := query(t, s, time.Hour, "BTC-USDT", "NONE-USDT")
	assert.Equal(t, []Point{{Time: hour, Value: "11"}, {Time: hour + time.Hour.Milliseconds(), Value: "23"}}, got["BTC-USDT"].Points)
	assert.Equal(t, Series{Points: []Point{}}, got["NONE-USDT"])
}

func TestCloseFlushesAndSurvivesReopen(t *testing.T) {
	cfg := testConfig(t)
	s, err := Open(context.Background(), cfg)
	require.NoError(t, err)
	s.Start(context.Background())
	s.Record("BTC-USDT", 7, "8", t0)
	require.NoError(t, s.Close())

	r := openStore(t, cfg)
	assert.Equal(t, []string{"7"}, values(query(t, r, 5*time.Minute, "BTC-USDT")["BTC-USDT"].Points))
}

func TestFlushLoop(t *testing.T) {
	old := FlushInterval
	FlushInterval = 10 * time.Millisecond
	t.Cleanup(func() { FlushInterval = old })

	s := openStore(t, testConfig(t))
	s.Start(context.Background())
	s.Record("BTC-USDT", 7, "8", t0)
	assert.Eventually(t, func() bool {
		return len(query(t, s, 5*time.Minute, "BTC-USDT")["BTC-USDT"].Points) == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestConfigFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DAL_SQLITE_PATH", filepath.Join(dir, "dal.db"))

	cfg, on, err := ConfigFromEnv()
	require.NoError(t, err)
	assert.True(t, on)
	assert.Equal(t, Config{Path: filepath.Join(dir, FileName), Interval: DefaultInterval, Retention: DefaultRetention}, cfg)

	t.Setenv(EnvPath, "/data/h.db")
	t.Setenv(EnvInterval, "1m")
	t.Setenv(EnvRetention, "48h")
	cfg, on, err = ConfigFromEnv()
	require.NoError(t, err)
	assert.True(t, on)
	assert.Equal(t, Config{Path: "/data/h.db", Interval: time.Minute, Retention: 48 * time.Hour}, cfg)

	t.Setenv(EnvRetention, "30s")
	_, _, err = ConfigFromEnv()
	assert.Error(t, err)
	t.Setenv(EnvRetention, "")

	t.Setenv(EnvInterval, "bogus")
	_, _, err = ConfigFromEnv()
	assert.Error(t, err)
	t.Setenv(EnvInterval, "")

	t.Setenv(EnvEnabled, "false")
	_, on, err = ConfigFromEnv()
	require.NoError(t, err)
	assert.False(t, on)

	t.Setenv(EnvEnabled, "")
	t.Setenv(EnvPath, "")
	t.Setenv("DAL_SQLITE_PATH", "")
	_, on, err = ConfigFromEnv()
	require.NoError(t, err)
	assert.False(t, on)
}

func values(points []Point) []string {
	out := make([]string, len(points))
	for i, p := range points {
		out[i] = p.Value
	}
	return out
}

func TestDecimalsChangeKeepsLatestScaleOnly(t *testing.T) {
	ctx := context.Background()
	s := openStore(t, testConfig(t))
	s.Record("BTC-USDT", 100000000, "8", t0)
	s.Record("BTC-USDT", 10000, "4", t0.Add(5*time.Minute))
	s.Record("BTC-USDT", 20000, "4", t0.Add(10*time.Minute))
	require.NoError(t, s.Flush(ctx))

	got := query(t, s, 5*time.Minute, "BTC-USDT")["BTC-USDT"]
	assert.Equal(t, "4", got.Decimals)
	assert.Equal(t, []string{"10000", "20000"}, values(got.Points))
}

func TestQueryStartsAtStepBoundary(t *testing.T) {
	ctx := context.Background()
	s := openStore(t, testConfig(t))
	hour := floorTo(t0.UnixMilli(), time.Hour.Milliseconds())
	s.Record("BTC-USDT", 1, "8", time.UnixMilli(hour+10*60_000)) // partial first step
	s.Record("BTC-USDT", 2, "8", time.UnixMilli(hour+70*60_000))
	require.NoError(t, s.Flush(ctx))

	got, err := s.Query(ctx, []string{"BTC-USDT"}, time.UnixMilli(hour+5*60_000), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, []Point{{Time: hour + time.Hour.Milliseconds(), Value: "2"}}, got["BTC-USDT"].Points)
}
