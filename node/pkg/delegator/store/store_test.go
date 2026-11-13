package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const proxy = "0xcb56b163e545a3870ca04c6ae2401f2405fb29a9"

func TestNormalizeEntry(t *testing.T) {
	contract, selector, err := NormalizeEntry("0xCB56B163E545A3870CA04C6AE2401F2405FB29A9", "submit(bytes32[],int256[],uint256[],bytes[])")
	require.NoError(t, err)
	assert.Equal(t, proxy, contract)
	assert.Regexp(t, `^0x[0-9a-f]{8}$`, selector)

	_, selector2, err := NormalizeEntry(proxy, strings.ToUpper(selector[2:]))
	assert.Error(t, err, "selector without 0x")
	_, selector2, err = NormalizeEntry(proxy, "0x"+strings.ToUpper(selector[2:]))
	require.NoError(t, err)
	assert.Equal(t, selector, selector2)

	_, _, err = NormalizeEntry("0x1234", selector)
	assert.Error(t, err)
	_, _, err = NormalizeEntry(proxy, "0x1234")
	assert.Error(t, err)
}

func TestAllowlist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	_, err := Open(ctx, dir, 0, false)
	assert.ErrorContains(t, err, "config.db", "missing file is an error without create")

	s, err := Open(ctx, dir, 0, true)
	require.NoError(t, err)
	defer s.Close()

	e, err := s.Upsert(ctx, proxy, "0xAABBCCDD", "first")
	require.NoError(t, err)
	assert.Equal(t, "0xaabbccdd", e.Selector)

	n, err := s.ImportCSV(ctx, strings.NewReader(proxy+",0x11223344,second\n"+proxy+",0xaabbccdd,renamed\n"))
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	_, err = s.ImportCSV(ctx, strings.NewReader(proxy+",bad\n"))
	assert.Error(t, err)

	entries, err := s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Entry{
		{Contract: proxy, Selector: "0x11223344", Description: "second"},
		{Contract: proxy, Selector: "0xaabbccdd", Description: "renamed"},
	}, entries)

	ok, err := s.ContractAllowed(ctx, strings.ToUpper(proxy[2:]))
	require.NoError(t, err)
	assert.False(t, ok, "contract lookups expect a 0x-prefixed lowercase address")
	ok, err = s.ContractAllowed(ctx, proxy)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = s.FunctionAllowed(ctx, proxy, "0x11223344")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = s.FunctionAllowed(ctx, proxy, "0xdeadbeef")
	require.NoError(t, err)
	assert.False(t, ok)

	removed, err := s.Remove(ctx, proxy, "0x11223344")
	require.NoError(t, err)
	assert.True(t, removed)
	removed, err = s.Remove(ctx, proxy, "0x11223344")
	require.NoError(t, err)
	assert.False(t, removed)
}

func TestRecordTransactionRotatesAndPrunes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir, 2, true)
	require.NoError(t, err)
	defer s.Close()

	// a stray non-matching file is never touched
	require.NoError(t, os.WriteFile(filepath.Join(dir, "transactions-notes.txt"), nil, 0o600))

	day := time.Date(2026, 11, 10, 23, 59, 0, 0, time.UTC)
	record := func(ts time.Time) {
		t.Helper()
		require.NoError(t, s.RecordTransaction(ctx, Transaction{Timestamp: ts, Sender: "0xA", Contract: proxy, Selector: "0xaabbccdd", Nonce: 1, TxHash: "0x01"}))
	}

	record(day)
	record(day.Add(2 * time.Minute)) // 11-11
	files, err := s.TransactionFiles()
	require.NoError(t, err)
	assert.Equal(t, []string{"transactions-20261110.db", "transactions-20261111.db"}, files)

	record(day.Add(24 * time.Hour)) // 11-11 again, same file
	record(day.Add(26 * time.Hour)) // 11-12: keeps 11-11 and 11-12
	files, err = s.TransactionFiles()
	require.NoError(t, err)
	assert.Equal(t, []string{"transactions-20261111.db", "transactions-20261112.db"}, files)

	_, err = os.Stat(filepath.Join(dir, "transactions-20261110.db-wal"))
	assert.True(t, os.IsNotExist(err), "wal of a deleted day is removed")
	_, err = os.Stat(filepath.Join(dir, "transactions-notes.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, configFile))
	assert.NoError(t, err, "config.db is never pruned")
}

func TestRetentionFromEnv(t *testing.T) {
	t.Setenv(EnvRetentionDays, "")
	days, err := RetentionFromEnv()
	require.NoError(t, err)
	assert.Equal(t, DefaultRetentionDays, days)

	t.Setenv(EnvRetentionDays, "0")
	days, err = RetentionFromEnv()
	require.NoError(t, err)
	assert.Equal(t, 0, days)

	t.Setenv(EnvRetentionDays, "-1")
	_, err = RetentionFromEnv()
	assert.Error(t, err)
}
