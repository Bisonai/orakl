package nodestate

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportFromPostgres runs against the migrated CI test database inside a
// transaction that is rolled back, so it leaves no rows behind.
func TestImportFromPostgres(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	encActive, addrActive := newEncryptedKey(t)
	encRetired, _ := newEncryptedKey(t)
	encLegacy, addrLegacy := newEncryptedKey(t)

	exec := func(sql string, args ...any) {
		t.Helper()
		_, err := tx.Exec(ctx, sql, args...)
		require.NoError(t, err)
	}
	exec("DELETE FROM signer_key")
	exec("DELETE FROM signer")
	exec("INSERT INTO signer_key (address, pk, state, created_at) VALUES (NULL, $1, 'retired', now() - interval '1 day')", encRetired)
	exec("INSERT INTO signer_key (address, pk, state, tx_hash) VALUES ($1, $2, 'active', '0xtx')", addrActive, encActive)
	exec("INSERT INTO signer (pk, unique_dummy) VALUES ($1, TRUE)", encLegacy)

	var cfgA, cfgB int32
	require.NoError(t, tx.QueryRow(ctx, "INSERT INTO configs (name) VALUES ('NODESTATE-TEST-A') RETURNING id").Scan(&cfgA))
	require.NoError(t, tx.QueryRow(ctx, "INSERT INTO configs (name) VALUES ('NODESTATE-TEST-B') RETURNING id").Scan(&cfgB))
	exec("INSERT INTO configs (name) VALUES ('NODESTATE-TEST-C')") // no rounds
	for _, r := range []int32{3, 10, 7} {
		exec("INSERT INTO global_aggregates (config_id, value, round) VALUES ($1, 1, $2)", cfgA, r)
	}
	exec("INSERT INTO global_aggregates (config_id, value, round) VALUES ($1, 1, 5)", cfgB)

	s, _ := openTemp(t)
	res, err := Import(ctx, tx, s, false)
	require.NoError(t, err)
	assert.False(t, res.Unchanged)
	assert.Empty(t, res.Warnings)

	got, err := s.Snapshot(ctx)
	require.NoError(t, err)

	// keyring in created_at order, then the legacy row (different key) as pending
	require.Len(t, got.Keys, 3)
	assert.Equal(t, encRetired, got.Keys[0].PK)
	assert.Equal(t, "retired", got.Keys[0].State)
	assert.Nil(t, got.Keys[0].Address)
	assert.Equal(t, encActive, got.Keys[1].PK)
	assert.Equal(t, "active", got.Keys[1].State)
	assert.Equal(t, addrActive, *got.Keys[1].Address)
	assert.Equal(t, "0xtx", *got.Keys[1].TxHash)
	assert.Equal(t, encLegacy, got.Keys[2].PK)
	assert.Equal(t, "pending", got.Keys[2].State)
	assert.Equal(t, addrLegacy, *got.Keys[2].Address)

	rounds := map[string]int32{}
	for _, r := range got.Rounds {
		rounds[r.ConfigName] = r.Round
	}
	assert.Equal(t, int32(10), rounds["NODESTATE-TEST-A"])
	assert.Equal(t, int32(5), rounds["NODESTATE-TEST-B"])
	assert.NotContains(t, rounds, "NODESTATE-TEST-C")

	// re-run: idempotent
	res, err = Import(ctx, tx, s, false)
	require.NoError(t, err)
	assert.True(t, res.Unchanged)

	// new round: refused without force, applied with force
	exec("INSERT INTO global_aggregates (config_id, value, round) VALUES ($1, 1, 11)", cfgA)
	_, err = Import(ctx, tx, s, false)
	assert.ErrorIs(t, err, ErrStoreNotEmpty)
	_, err = Import(ctx, tx, s, true)
	require.NoError(t, err)
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	for _, r := range got.Rounds {
		if r.ConfigName == "NODESTATE-TEST-A" {
			assert.Equal(t, int32(11), r.Round)
		}
	}

	// legacy row mirrored into keyring (same key, re-encrypted) is not duplicated
	exec("DELETE FROM signer")
	exec("INSERT INTO signer (pk, unique_dummy) VALUES ($1, TRUE)", reEncrypt(t, encActive))
	_, err = Import(ctx, tx, s, true)
	require.NoError(t, err)
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	assert.Len(t, got.Keys, 2)
}
