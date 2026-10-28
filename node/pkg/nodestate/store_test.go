package nodestate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "node.db")
	s, err := Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func strPtr(s string) *string { return &s }

func sameContent(a, b Snapshot) bool {
	return sameKeys(a.Keys, b.Keys) && sameRounds(a.Rounds, b.Rounds)
}

func sampleSnapshot() Snapshot {
	t0 := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	return Snapshot{
		Keys: []utils.SignerKey{
			{Address: strPtr("0xaaa"), PK: "enc-old", State: "retired", CreatedAt: t0, UpdatedAt: t0.Add(time.Hour)},
			{Address: strPtr("0xbbb"), PK: "enc-active", State: "active", TxHash: strPtr("0xtx"), CreatedAt: t0.Add(time.Minute), UpdatedAt: t0.Add(2 * time.Hour)},
			{PK: "enc-noaddr", State: "pending", CreatedAt: t0.Add(2 * time.Minute), UpdatedAt: t0.Add(2 * time.Minute)},
		},
		Rounds: []Round{{ConfigName: "BTC-USDT", Round: 42}, {ConfigName: "ETH-USDT", Round: 7}},
	}
}

func TestReplaceRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)

	empty, err := s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, empty.IsEmpty())

	want := sampleSnapshot()
	require.NoError(t, s.Replace(ctx, want))

	got, err := s.Snapshot(ctx)
	require.NoError(t, err)
	require.Len(t, got.Keys, 3)
	for i := range want.Keys {
		assert.Equal(t, want.Keys[i].PK, got.Keys[i].PK)
		assert.Equal(t, want.Keys[i].State, got.Keys[i].State)
		assert.Equal(t, want.Keys[i].Address, got.Keys[i].Address)
		assert.Equal(t, want.Keys[i].TxHash, got.Keys[i].TxHash)
		assert.True(t, want.Keys[i].CreatedAt.Equal(got.Keys[i].CreatedAt))
		assert.True(t, want.Keys[i].UpdatedAt.Equal(got.Keys[i].UpdatedAt))
	}
	assert.Equal(t, want.Rounds, got.Rounds)
	assert.True(t, sameContent(want, got))

	// replace drops old content
	require.NoError(t, s.Replace(ctx, Snapshot{Rounds: []Round{{ConfigName: "X", Round: 1}}}))
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	assert.Empty(t, got.Keys)
	assert.Equal(t, []Round{{ConfigName: "X", Round: 1}}, got.Rounds)
}

func TestReplaceRejectsTwoActiveAtomically(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	require.NoError(t, s.Replace(ctx, sampleSnapshot()))

	now := time.Now()
	bad := Snapshot{Keys: []utils.SignerKey{
		{PK: "a", State: "active", CreatedAt: now, UpdatedAt: now},
		{PK: "b", State: "active", CreatedAt: now, UpdatedAt: now},
	}}
	assert.Error(t, s.Replace(ctx, bad))

	// previous content kept
	got, err := s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(sampleSnapshot(), got))
}

func TestReplaceRejectsUnknownState(t *testing.T) {
	s, _ := openTemp(t)
	now := time.Now()
	err := s.Replace(context.Background(), Snapshot{Keys: []utils.SignerKey{{PK: "a", State: "legacy", CreatedAt: now, UpdatedAt: now}}})
	assert.Error(t, err)
}

func TestReopenKeepsData(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	require.NoError(t, s.Replace(ctx, sampleSnapshot()))
	require.NoError(t, s.Close())

	s2, err := Open(ctx, path)
	require.NoError(t, err)
	defer s2.Close()
	got, err := s2.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(sampleSnapshot(), got))
}

func TestOpenFromEnv(t *testing.T) {
	ctx := context.Background()
	t.Setenv(EnvPath, "")
	_, err := OpenFromEnv(ctx, false)
	assert.ErrorContains(t, err, EnvPath)

	path := filepath.Join(t.TempDir(), "node.db")
	t.Setenv(EnvPath, path)
	_, err = OpenFromEnv(ctx, true)
	assert.Error(t, err, "mustExist on missing file")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))

	s, err := OpenFromEnv(ctx, false)
	require.NoError(t, err)
	require.NoError(t, s.Close())

	s, err = OpenFromEnv(ctx, true)
	require.NoError(t, err)
	require.NoError(t, s.Close())
}

func TestOpenRejectsURIChars(t *testing.T) {
	for _, p := range []string{"a?b.db", "a#b.db", "a%20b.db"} {
		_, err := Open(context.Background(), filepath.Join(t.TempDir(), p))
		assert.Error(t, err, p)
	}
}

func TestKeyringLifecycle(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)

	a, err := s.InsertPendingKey(ctx, "0xaaa", "enc-a")
	require.NoError(t, err)
	require.NoError(t, s.ActivateKey(ctx, a))
	require.NoError(t, s.ActivateKey(ctx, a), "re-activating the active key is a no-op")

	b, err := s.InsertPendingKey(ctx, "0xbbb", "enc-b")
	require.NoError(t, err)
	_, err = s.InsertPendingKey(ctx, "0xBBB", "enc-dup")
	assert.Error(t, err, "address is unique, case-insensitive")

	require.NoError(t, s.ActivateKey(ctx, b))
	assert.Error(t, s.ActivateKey(ctx, 999), "unknown id")

	// reopen: b active, a retired, unknown-id activate changed nothing
	require.NoError(t, s.Close())
	s, err = Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	keys, err := s.SignerKeys(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "retired", keys[0].State)
	assert.Equal(t, "active", keys[1].State)
	assert.Equal(t, "enc-b", keys[1].PK)
}

func TestBackfillKeyAddress(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	now := time.Now()
	require.NoError(t, s.Replace(ctx, Snapshot{Keys: []utils.SignerKey{
		{PK: "enc", State: "active", CreatedAt: now, UpdatedAt: now},
	}}))
	keys, err := s.SignerKeys(ctx)
	require.NoError(t, err)

	require.NoError(t, s.BackfillKeyAddress(ctx, keys[0].ID, "0xabc"))
	require.NoError(t, s.BackfillKeyAddress(ctx, keys[0].ID, "0xdef"), "only a NULL address is set")
	keys, err = s.SignerKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0xabc", *keys[0].Address)
}

func TestGCRetiredKeys(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	old := time.Now().Add(-100 * 24 * time.Hour)
	recent := time.Now().Add(-time.Hour)
	require.NoError(t, s.Replace(ctx, Snapshot{Keys: []utils.SignerKey{
		{Address: strPtr("0x1"), PK: "old-retired", State: "retired", CreatedAt: old, UpdatedAt: old},
		{Address: strPtr("0x2"), PK: "recent-retired", State: "retired", CreatedAt: old, UpdatedAt: recent},
		{Address: strPtr("0x3"), PK: "old-pending", State: "pending", CreatedAt: old, UpdatedAt: old},
		{Address: strPtr("0x4"), PK: "active", State: "active", CreatedAt: old, UpdatedAt: old},
	}}))

	require.NoError(t, s.GCRetiredKeys(ctx, time.Now().Add(-90*24*time.Hour)))
	keys, err := s.SignerKeys(ctx)
	require.NoError(t, err)
	var pks []string
	for _, k := range keys {
		pks = append(pks, k.PK)
	}
	assert.Equal(t, []string{"recent-retired", "old-pending", "active"}, pks)
}

func TestSaveRoundsNeverGoesDown(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)

	_, ok, err := s.LatestRound(ctx, "BTC-USDT")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, s.SaveRounds(ctx, map[string]int32{"BTC-USDT": 10, "ETH-USDT": 5}))
	require.NoError(t, s.SaveRounds(ctx, map[string]int32{"BTC-USDT": 7, "ETH-USDT": 6}))

	require.NoError(t, s.Close())
	s, err = Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	r, ok, err := s.LatestRound(ctx, "BTC-USDT")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int32(10), r)
	r, _, err = s.LatestRound(ctx, "ETH-USDT")
	require.NoError(t, err)
	assert.Equal(t, int32(6), r)
}
