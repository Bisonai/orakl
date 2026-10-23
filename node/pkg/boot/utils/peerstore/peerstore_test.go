package peerstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "boot.db")
	s, err := Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestInsertListDelete(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)

	peers, err := s.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, peers)
	assert.Empty(t, peers)

	p1, inserted, err := s.Insert(ctx, "/ip4/127.0.0.1/tcp/10001/p2p/a")
	require.NoError(t, err)
	assert.True(t, inserted)
	assert.NotZero(t, p1.ID)
	assert.Equal(t, "/ip4/127.0.0.1/tcp/10001/p2p/a", p1.Url)

	p2, inserted, err := s.Insert(ctx, "/ip4/127.0.0.1/tcp/10002/p2p/b")
	require.NoError(t, err)
	assert.True(t, inserted)

	// duplicate url is a no-op: zero peer, no error
	dup, inserted, err := s.Insert(ctx, "/ip4/127.0.0.1/tcp/10001/p2p/a")
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, Peer{}, dup)

	peers, err = s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Peer{p1, p2}, peers)

	removed, err := s.DeleteByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.True(t, removed)
	removed, err = s.DeleteByID(ctx, p1.ID)
	require.NoError(t, err)
	assert.False(t, removed)

	peers, err = s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Peer{p2}, peers)
}

func TestInsertEmptyURL(t *testing.T) {
	s, _ := openTemp(t)
	_, _, err := s.Insert(context.Background(), "")
	assert.Error(t, err)
}

func TestEmptyListEncodesAsArray(t *testing.T) {
	s, _ := openTemp(t)
	peers, err := s.List(context.Background())
	require.NoError(t, err)
	b, err := json.Marshal(peers)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(b))
}

func TestPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	p, _, err := s.Insert(ctx, "/ip4/127.0.0.1/tcp/10001/p2p/a")
	require.NoError(t, err)
	require.NoError(t, s.Close())

	s2, err := Open(ctx, path)
	require.NoError(t, err)
	defer s2.Close()

	peers, err := s2.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Peer{p}, peers)
}

func TestImportURLs(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	_, _, err := s.Insert(ctx, "/ip4/127.0.0.1/tcp/10001/p2p/a")
	require.NoError(t, err)

	n, err := s.ImportURLs(ctx, strings.NewReader("/ip4/127.0.0.1/tcp/10001/p2p/a\n\n  /ip4/127.0.0.1/tcp/10002/p2p/b  \n/ip4/127.0.0.1/tcp/10003/p2p/c"))
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	peers, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, peers, 3)
	assert.Equal(t, "/ip4/127.0.0.1/tcp/10002/p2p/b", peers[1].Url)
}

func TestOpenRejectsURIChars(t *testing.T) {
	for _, p := range []string{"a?b.db", "a#b.db", "a%41.db"} {
		_, err := Open(context.Background(), filepath.Join(t.TempDir(), p))
		assert.Error(t, err, p)
	}
}

func TestOpenFromEnv(t *testing.T) {
	ctx := context.Background()

	t.Setenv(EnvPath, "")
	_, err := OpenFromEnv(ctx, false)
	assert.ErrorIs(t, err, ErrPathNotSet)

	path := filepath.Join(t.TempDir(), "boot.db")
	t.Setenv(EnvPath, path)
	_, err = OpenFromEnv(ctx, true)
	assert.ErrorIs(t, err, os.ErrNotExist)

	// missing file is created empty: peers self-register via /peer/sync
	s, err := OpenFromEnv(ctx, false)
	require.NoError(t, err)
	require.NoError(t, s.Ping(ctx))
	require.NoError(t, s.Close())
	_, err = os.Stat(path)
	require.NoError(t, err)

	s, err = OpenFromEnv(ctx, true)
	require.NoError(t, err)
	require.NoError(t, s.Close())
}
