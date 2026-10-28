package nodestate

import (
	"context"
	"strings"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/utils/encryptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEncryptedKey returns (encrypted pk, lowercased address) of a fresh key.
func newEncryptedKey(t *testing.T) (string, string) {
	t.Helper()
	_, pkHex, err := utils.NewPk(context.Background())
	require.NoError(t, err)
	addr, err := utils.StringPkToAddressHex(pkHex)
	require.NoError(t, err)
	enc, err := encryptor.EncryptText(pkHex)
	require.NoError(t, err)
	return enc, strings.ToLower(addr)
}

// reEncrypt returns enc's key encrypted again (new random IV/salt).
func reEncrypt(t *testing.T, enc string) string {
	t.Helper()
	pk, err := encryptor.DecryptText(enc)
	require.NoError(t, err)
	reEnc, err := encryptor.EncryptText(pk)
	require.NoError(t, err)
	return reEnc
}

func TestLegacyKey(t *testing.T) {
	t.Setenv("ENCRYPT_PASSWORD", "anything")
	now := time.Now().UTC()

	encA, addrA := newEncryptedKey(t)
	encB, addrB := newEncryptedKey(t)

	t.Run("empty keyring -> active with address", func(t *testing.T) {
		k, err := legacyKey(nil, encA, now)
		require.NoError(t, err)
		require.NotNil(t, k)
		assert.Equal(t, "active", k.State)
		assert.Equal(t, encA, k.PK)
		require.NotNil(t, k.Address)
		assert.Equal(t, addrA, *k.Address)
	})

	t.Run("same ciphertext in keyring -> skipped", func(t *testing.T) {
		k, err := legacyKey([]utils.SignerKey{{PK: encA, State: "active"}}, encA, now)
		require.NoError(t, err)
		assert.Nil(t, k)
	})

	t.Run("re-encrypted same key, keyring address set -> skipped", func(t *testing.T) {
		reEnc := reEncrypt(t, encA)
		require.NotEqual(t, encA, reEnc)
		upper := strings.ToUpper(addrA[2:])
		k, err := legacyKey([]utils.SignerKey{{Address: strPtr("0x" + upper), PK: encA, State: "active"}}, reEnc, now)
		require.NoError(t, err)
		assert.Nil(t, k)
	})

	t.Run("re-encrypted same key, keyring address NULL -> skipped", func(t *testing.T) {
		reEnc := reEncrypt(t, encA)
		k, err := legacyKey([]utils.SignerKey{{PK: encA, State: "active"}}, reEnc, now)
		require.NoError(t, err)
		assert.Nil(t, k)
	})

	t.Run("different key with active keyring -> pending", func(t *testing.T) {
		k, err := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "active"}}, encB, now)
		require.NoError(t, err)
		require.NotNil(t, k)
		assert.Equal(t, "pending", k.State)
		assert.Equal(t, addrB, *k.Address)
	})

	t.Run("different key with pending-only keyring -> pending", func(t *testing.T) {
		k, err := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "pending"}}, encB, now)
		require.NoError(t, err)
		require.NotNil(t, k)
		assert.Equal(t, "pending", k.State)
	})

	t.Run("undecryptable -> error without key material", func(t *testing.T) {
		_, err := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "retired"}}, "not-a-ciphertext", now)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not-a-ciphertext")
	})

	t.Run("wrong password -> error", func(t *testing.T) {
		t.Setenv("ENCRYPT_PASSWORD", "wrong")
		_, err := legacyKey([]utils.SignerKey{{PK: encA, State: "retired"}}, encB, now)
		assert.Error(t, err)
	})
}

func TestImportSnapshot(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	snap := sampleSnapshot()

	res, err := Apply(ctx, s, snap, false)
	require.NoError(t, err)
	assert.False(t, res.Unchanged)

	// same content again: idempotent no-op, even if timestamps differ
	again := sampleSnapshot()
	again.Keys[2].CreatedAt = time.Now()
	res, err = Apply(ctx, s, again, false)
	require.NoError(t, err)
	assert.True(t, res.Unchanged)

	// only rounds advanced: updated without force
	newer := sampleSnapshot()
	newer.Rounds[0].Round = 43
	res, err = Apply(ctx, s, newer, false)
	require.NoError(t, err)
	assert.True(t, res.RoundsOnly)
	got, err := s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(newer, got))

	// rounds regress or a config disappears: refused without force
	older := sampleSnapshot()
	_, err = Apply(ctx, s, older, false)
	assert.ErrorIs(t, err, ErrStoreDiffers)
	dropped := sampleSnapshot()
	dropped.Rounds = []Round{{ConfigName: "BTC-USDT", Round: 43}}
	_, err = Apply(ctx, s, dropped, false)
	assert.ErrorIs(t, err, ErrStoreDiffers)

	// rounds-only update keeps key rows (ids) untouched
	before, err := s.SignerKeys(ctx)
	require.NoError(t, err)
	newest := sampleSnapshot()
	newest.Rounds[0].Round = 44
	newest.Rounds = append(newest.Rounds, Round{ConfigName: "NEW", Round: 1})
	res, err = Apply(ctx, s, newest, false)
	require.NoError(t, err)
	assert.True(t, res.RoundsOnly)
	after, err := s.SignerKeys(ctx)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	newer = newest

	// different keyring: refused without force, store untouched
	changed := sampleSnapshot()
	changed.Keys[0].State = "pending"
	_, err = Apply(ctx, s, changed, false)
	assert.ErrorIs(t, err, ErrStoreDiffers)
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(newer, got))

	// force replaces
	res, err = Apply(ctx, s, changed, true)
	require.NoError(t, err)
	assert.False(t, res.Unchanged)
	assert.False(t, res.RoundsOnly)
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(changed, got))
}
