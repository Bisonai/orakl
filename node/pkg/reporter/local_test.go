package reporter

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	chainutils "bisonai.com/miko/node/pkg/chain/utils"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestKeepsNewest(t *testing.T) {
	a := NewNodeApp()
	ts := time.UnixMilli(1_700_000_000_000)

	a.Ingest("A-B", 1, ts, []byte{1}, true)
	a.Ingest("A-B", 2, ts.Add(-time.Second), []byte{2}, false)

	got, ok := GetLatestData(a.LatestDataMap, "A-B")
	require.True(t, ok)
	assert.Equal(t, int64(1), got.Value)
	assert.Equal(t, ts.UnixMilli(), got.AggregateTime)
	assert.Equal(t, [32]byte(crypto.Keccak256([]byte("A-B"))), got.FeedHash)
	leader, _ := a.local.leaders.Load("A-B")
	assert.Equal(t, true, leader, "older round must not change leadership")

	a.Ingest("A-B", 3, ts.Add(time.Second), []byte{3}, false)
	got, _ = GetLatestData(a.LatestDataMap, "A-B")
	assert.Equal(t, int64(3), got.Value)
	leader, _ = a.local.leaders.Load("A-B")
	assert.Equal(t, false, leader)
}

func TestIngestIgnoredWithoutLocal(t *testing.T) {
	a := New()
	a.Ingest("A-B", 1, time.Now(), []byte{1}, true)
	_, ok := a.LatestDataMap.Load("A-B")
	assert.False(t, ok)
}

func TestLocalPrepare(t *testing.T) {
	now := time.UnixMilli(1_700_000_000_000)
	pkA, err := crypto.GenerateKey()
	require.NoError(t, err)
	pkB, err := crypto.GenerateKey()
	require.NoError(t, err)
	addrA := crypto.PubkeyToAddress(pkA.PublicKey)
	addrB := crypto.PubkeyToAddress(pkB.PublicKey)

	sig := func(pk *ecdsa.PrivateKey, value int64, ts time.Time, symbol string) []byte {
		s, err := chainutils.MakeValueSignature(value, ts.UnixMilli(), symbol, pk)
		require.NoError(t, err)
		return s
	}

	a := NewNodeApp()
	a.local.now = func() time.Time { return now }
	a.local.setWhitelist([]kaiacommon.Address{addrB, addrA})

	fresh := now.Add(-time.Second)
	stale := now.Add(-LocalMaxAge - time.Second)
	a.Ingest("LED", 10, fresh, bytes.Join([][]byte{sig(pkA, 10, fresh, "LED"), sig(pkB, 10, fresh, "LED")}, nil), true)
	a.Ingest("FOLLOWED", 20, fresh, sig(pkA, 20, fresh, "FOLLOWED"), false)
	a.Ingest("STALE", 30, stale, sig(pkA, 30, stale, "STALE"), true)
	a.Ingest("BADPROOF", 40, fresh, sig(pkA, 41, fresh, "BADPROOF"), true)

	pairs := []string{"LED", "FOLLOWED", "STALE", "BADPROOF", "MISSING"}
	got := a.local.prepare(context.Background(), a.local.latest(a.LatestDataMap, pairs))

	require.Len(t, got, 1)
	led, ok := got["LED"]
	require.True(t, ok)
	assert.Equal(t, int64(10), led.Value)
	assert.Equal(t, bytes.Join([][]byte{sig(pkB, 10, fresh, "LED"), sig(pkA, 10, fresh, "LED")}, nil), led.Proof, "proof in whitelist order")

	stored, _ := GetLatestData(a.LatestDataMap, "LED")
	assert.Equal(t, bytes.Join([][]byte{sig(pkA, 10, fresh, "LED"), sig(pkB, 10, fresh, "LED")}, nil), stored.Proof, "stored proof untouched")
}
