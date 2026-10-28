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
