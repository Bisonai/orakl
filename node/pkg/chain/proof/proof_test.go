package proof

import (
	"bytes"
	"crypto/ecdsa"
	"testing"
	"time"

	chainutils "bisonai.com/miko/node/pkg/chain/utils"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKey(t *testing.T) (*ecdsa.PrivateKey, kaiacommon.Address) {
	t.Helper()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	return pk, crypto.PubkeyToAddress(pk.PublicKey)
}

func sign(t *testing.T, pk *ecdsa.PrivateKey, value int64, ts time.Time, symbol string) []byte {
	t.Helper()
	sig, err := chainutils.MakeValueSignature(value, ts.UnixMilli(), symbol, pk)
	require.NoError(t, err)
	return sig
}

func TestOrder(t *testing.T) {
	ts := time.UnixMilli(1_700_000_000_000)
	pkA, addrA := newKey(t)
	pkB, addrB := newKey(t)
	sigA := sign(t, pkA, 42, ts, "BTC-USDT")
	sigB := sign(t, pkB, 42, ts, "BTC-USDT")

	t.Run("whitelist order, duplicates dropped", func(t *testing.T) {
		raw := bytes.Join([][]byte{sigA, sigB, sigA}, nil)
		got, err := Order(raw, 42, ts, "BTC-USDT", []kaiacommon.Address{addrB, addrA})
		require.NoError(t, err)
		assert.Equal(t, bytes.Join([][]byte{sigB, sigA}, nil), got)
	})

	t.Run("non-whitelisted signer", func(t *testing.T) {
		_, err := Order(bytes.Join([][]byte{sigA, sigB}, nil), 42, ts, "BTC-USDT", []kaiacommon.Address{addrA})
		assert.ErrorIs(t, err, errorsentinel.ErrDalSignerNotWhitelisted)
	})

	t.Run("signature over other value", func(t *testing.T) {
		_, err := Order(sigA, 43, ts, "BTC-USDT", []kaiacommon.Address{addrA})
		assert.ErrorIs(t, err, errorsentinel.ErrDalSignerNotWhitelisted)
	})

	t.Run("invalid length", func(t *testing.T) {
		_, err := Order(sigA[:64], 42, ts, "BTC-USDT", []kaiacommon.Address{addrA})
		assert.ErrorIs(t, err, errorsentinel.ErrDalInvalidProofLength)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := Order(nil, 42, ts, "BTC-USDT", []kaiacommon.Address{addrA})
		assert.ErrorIs(t, err, errorsentinel.ErrDalEmptyProofParam)
	})
}
