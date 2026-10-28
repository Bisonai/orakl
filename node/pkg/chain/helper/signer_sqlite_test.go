package helper

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/nodestate"
	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Rotation lifecycle against the real SQLite keyring (fake chain): the keyring must survive a
// store reopen / new Signer, and an interrupted rotation must resume its pending key.

func openSignerState(t *testing.T, path string) *nodestate.Store {
	t.Helper()
	st, err := nodestate.Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func keyStates(t *testing.T, st *nodestate.Store) map[string]string {
	t.Helper()
	keys, err := st.SignerKeys(context.Background())
	require.NoError(t, err)
	out := map[string]string{}
	for _, k := range keys {
		require.NotNil(t, k.Address)
		out[*k.Address] = k.State
	}
	return out
}

func toLowerHex(a common.Address) string { return strings.ToLower(a.Hex()) }

func TestSQLiteSignerRotateReopenKeepsActiveKey(t *testing.T) {
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")

	aHex, aAddr, aPK := genKey(t)
	st := openSignerState(t, path)
	store := sqliteSignerStore{state: st}
	id, err := store.InsertPending(ctx, aAddr.Hex(), aHex)
	require.NoError(t, err)
	require.NoError(t, store.Promote(ctx, id))

	// pk is encrypted at rest
	raw, err := st.SignerKeys(ctx)
	require.NoError(t, err)
	assert.NotContains(t, raw[0].PK, aHex)

	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour) // near expiry -> rotate
	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	require.NoError(t, s.reconcile(ctx, true))
	newAddr := s.activeAddr
	require.NotEqual(t, aAddr, newAddr, "expected a rotation")
	require.True(t, s.usable)
	chain.setDeactivated(aAddr) // updateOracle deactivates the old key on-chain

	states := keyStates(t, st)
	require.Len(t, states, 2)
	assert.Equal(t, "retired", states[toLowerHex(aAddr)])
	assert.Equal(t, "active", states[toLowerHex(newAddr)])

	// restart: reopen the store and build a fresh Signer
	require.NoError(t, st.Close())
	st2 := openSignerState(t, path)
	s2 := newScenarioSigner(chain, sqliteSignerStore{state: st2})
	require.NoError(t, s2.reconcile(ctx, false))
	assert.Equal(t, newAddr, s2.activeAddr)
	assert.True(t, s2.usable)
	assert.Len(t, keyStates(t, st2), 2, "reconcile must not add keys")
}

func TestSQLiteSignerInterruptedRotationResumes(t *testing.T) {
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")

	aHex, aAddr, aPK := genKey(t)
	st := openSignerState(t, path)
	store := sqliteSignerStore{state: st}
	id, err := store.InsertPending(ctx, aAddr.Hex(), aHex)
	require.NoError(t, err)
	require.NoError(t, store.Promote(ctx, id))

	// first attempt: updateOracle never lands -> pending key persisted, old key kept
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour)
	chain.updateOracle = func(*fakeChain, string, common.Address) error { return nil }
	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)
	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, aAddr, s.activeAddr)

	states := keyStates(t, st)
	require.Len(t, states, 2)
	var pending string
	for addr, state := range states {
		if state == "pending" {
			pending = addr
		}
	}
	require.NotEmpty(t, pending, "expected a persisted pending key")

	// restart, chain now mines: the pending key is resumed, no new key minted
	require.NoError(t, st.Close())
	st2 := openSignerState(t, path)
	chain.updateOracle = nil
	s2 := newScenarioSigner(chain, sqliteSignerStore{state: st2})
	require.NoError(t, s2.reconcile(ctx, false)) // adopts the whitelisted old key
	require.Equal(t, aAddr, s2.activeAddr)
	require.NoError(t, s2.reconcile(ctx, true)) // rotates, reusing the pending key

	assert.Equal(t, pending, toLowerHex(s2.activeAddr))
	states = keyStates(t, st2)
	assert.Len(t, states, 2, "no duplicate key")
	assert.Equal(t, "active", states[pending])
	assert.Equal(t, "retired", states[toLowerHex(aAddr)])
}

func TestSQLiteEnsureKeyringSeedsOnce(t *testing.T) {
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	ctx := context.Background()
	st := openSignerState(t, filepath.Join(t.TempDir(), "node.db"))
	store := sqliteSignerStore{state: st}

	seedHex, seedAddr, _ := genKey(t)
	t.Setenv(SignerPk, "0x"+seedHex)

	pk, err := ensureKeyring(ctx, store, SignerConfig{})
	require.NoError(t, err)
	assert.Equal(t, seedHex, pk)
	assert.Equal(t, map[string]string{toLowerHex(seedAddr): "active"}, keyStates(t, st))

	// a non-empty keyring is never re-seeded
	otherHex, _, _ := genKey(t)
	t.Setenv(SignerPk, otherHex)
	pk, err = ensureKeyring(ctx, store, SignerConfig{})
	require.NoError(t, err)
	assert.Equal(t, seedHex, pk)
	assert.Len(t, keyStates(t, st), 1)
}

func TestNewSignerRequiresStore(t *testing.T) {
	_, err := NewSigner(context.Background())
	assert.Error(t, err)
}

// With a wrong ENCRYPT_PASSWORD no key decrypts: init must fail, not seed
// SIGNER_PK over (and retire) the live keyring.
func TestSQLiteEnsureKeyringUndecryptableFails(t *testing.T) {
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	ctx := context.Background()
	st := openSignerState(t, filepath.Join(t.TempDir(), "node.db"))
	store := sqliteSignerStore{state: st}
	liveHex, liveAddr, _ := genKey(t)
	id, err := store.InsertPending(ctx, liveAddr.Hex(), liveHex)
	require.NoError(t, err)
	require.NoError(t, store.Promote(ctx, id))

	t.Setenv("ENCRYPT_PASSWORD", "wrong")
	seedHex, _, _ := genKey(t)
	t.Setenv(SignerPk, seedHex)
	_, err = ensureKeyring(ctx, store, SignerConfig{})
	assert.Error(t, err)
	assert.Equal(t, map[string]string{toLowerHex(liveAddr): "active"}, keyStates(t, st))
}
