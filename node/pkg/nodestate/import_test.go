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
		k, warning := legacyKey(nil, encA, now)
		require.NotNil(t, k)
		assert.Empty(t, warning)
		assert.Equal(t, "active", k.State)
		assert.Equal(t, encA, k.PK)
		require.NotNil(t, k.Address)
		assert.Equal(t, addrA, *k.Address)
	})

	t.Run("same ciphertext in keyring -> skipped", func(t *testing.T) {
		k, _ := legacyKey([]utils.SignerKey{{PK: encA, State: "active"}}, encA, now)
		assert.Nil(t, k)
	})

	t.Run("re-encrypted same key, keyring address set -> skipped", func(t *testing.T) {
		reEnc := reEncrypt(t, encA)
		require.NotEqual(t, encA, reEnc)
		upper := strings.ToUpper(addrA[2:])
		k, _ := legacyKey([]utils.SignerKey{{Address: strPtr("0x" + upper), PK: encA, State: "active"}}, reEnc, now)
		assert.Nil(t, k)
	})

	t.Run("re-encrypted same key, keyring address NULL -> skipped", func(t *testing.T) {
		reEnc := reEncrypt(t, encA)
		k, _ := legacyKey([]utils.SignerKey{{PK: encA, State: "active"}}, reEnc, now)
		assert.Nil(t, k)
	})

	t.Run("different key with active keyring -> pending", func(t *testing.T) {
		k, warning := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "active"}}, encB, now)
		require.NotNil(t, k)
		assert.Empty(t, warning)
		assert.Equal(t, "pending", k.State)
		assert.Equal(t, addrB, *k.Address)
	})

	t.Run("undecryptable -> kept without address + warning", func(t *testing.T) {
		k, warning := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "retired"}}, "not-a-ciphertext", now)
		require.NotNil(t, k)
		assert.Nil(t, k.Address)
		assert.Equal(t, "active", k.State)
		assert.NotEmpty(t, warning)
		assert.NotContains(t, warning, "not-a-ciphertext")
	})

	t.Run("wrong password -> kept without address", func(t *testing.T) {
		t.Setenv("ENCRYPT_PASSWORD", "wrong")
		k, warning := legacyKey([]utils.SignerKey{{Address: &addrA, PK: encA, State: "active"}}, encB, now)
		require.NotNil(t, k)
		assert.Nil(t, k.Address)
		assert.NotEmpty(t, warning)
	})
}

func TestImportSnapshot(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	snap := sampleSnapshot()

	res, err := importSnapshot(ctx, s, snap, false)
	require.NoError(t, err)
	assert.False(t, res.Unchanged)

	// same content again: idempotent no-op, even if timestamps differ
	again := sampleSnapshot()
	again.Keys[2].CreatedAt = time.Now()
	res, err = importSnapshot(ctx, s, again, false)
	require.NoError(t, err)
	assert.True(t, res.Unchanged)

	// different content: refused without force, store untouched
	changed := sampleSnapshot()
	changed.Rounds[0].Round = 43
	_, err = importSnapshot(ctx, s, changed, false)
	assert.ErrorIs(t, err, ErrStoreNotEmpty)
	got, err := s.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(42), got.Rounds[0].Round)

	// force replaces
	res, err = importSnapshot(ctx, s, changed, true)
	require.NoError(t, err)
	assert.False(t, res.Unchanged)
	got, err = s.Snapshot(ctx)
	require.NoError(t, err)
	assert.True(t, sameContent(changed, got))
}
