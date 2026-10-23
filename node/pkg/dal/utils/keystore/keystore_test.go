package keystore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dal.db")
	s, err := Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestUpsertLookupRemove(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)

	_, ok, err := s.Lookup(ctx, "k1")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, s.Upsert(ctx, "k1", "alice"))
	desc, ok, err := s.Lookup(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "alice", desc)

	require.NoError(t, s.Upsert(ctx, "k1", "alice-2"))
	desc, _, err = s.Lookup(ctx, "k1")
	require.NoError(t, err)
	assert.Equal(t, "alice-2", desc)

	n, err := s.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	removed, err := s.Remove(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, removed)
	removed, err = s.Remove(ctx, "k1")
	require.NoError(t, err)
	assert.False(t, removed)

	_, ok, err = s.Lookup(ctx, "k1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestUpsertEmptyKey(t *testing.T) {
	s, _ := openTemp(t)
	assert.Error(t, s.Upsert(context.Background(), "", "x"))
}

func TestPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	require.NoError(t, s.Upsert(ctx, "k1", "alice"))
	require.NoError(t, s.Close())

	s2, err := Open(ctx, path)
	require.NoError(t, err)
	defer s2.Close()

	desc, ok, err := s2.Lookup(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "alice", desc)
}

func TestImportCSV(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	require.NoError(t, s.Upsert(ctx, "k1", "old"))

	in := "k1,alice\nk2,\"bob, inc\"\nk3,\nk4\n"
	n, err := s.ImportCSV(ctx, strings.NewReader(in))
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	keys, err := s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Key{
		{Key: "k1", Description: "alice"},
		{Key: "k2", Description: "bob, inc"},
		{Key: "k3", Description: ""},
		{Key: "k4", Description: ""},
	}, keys)
}

func TestImportCSVRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)

	_, err := s.ImportCSV(ctx, strings.NewReader("k1,alice\n,missing-key\n"))
	assert.Error(t, err)

	n, err := s.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestOpenRejectsURIChars(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"dal?x.db", "dal#x.db", "dal%20x.db"} {
		_, err := Open(context.Background(), filepath.Join(dir, name))
		assert.Error(t, err, name)
	}
}

func TestOpenFromEnv(t *testing.T) {
	ctx := context.Background()
	t.Setenv(EnvPath, "")
	_, err := OpenFromEnv(ctx, false)
	assert.ErrorIs(t, err, ErrPathNotSet)

	t.Setenv(EnvPath, filepath.Join(t.TempDir(), "dal.db"))
	_, err = OpenFromEnv(ctx, true)
	assert.ErrorIs(t, err, os.ErrNotExist, "mustExist must not create the file")

	s, err := OpenFromEnv(ctx, false)
	require.NoError(t, err)
	require.NoError(t, s.Close())

	s, err = OpenFromEnv(ctx, true)
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}
