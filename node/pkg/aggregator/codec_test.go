//nolint:all
package aggregator

import (
	"bytes"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/raft"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPeerID(t *testing.T) string {
	t.Helper()
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	require.NoError(t, err)
	id, err := peer.IDFromPrivateKey(priv)
	require.NoError(t, err)
	return id.String()
}

func TestAggregatorInnerRoundTripBothFormats(t *testing.T) {
	pid := testPeerID(t)
	ts := time.Now()

	trigger := TriggerMessage{LeaderID: pid, RoundID: 42, Timestamp: ts}
	proof := ProofMessage{RoundID: 7, Value: 12345, Proof: []byte{0xde, 0xad, 0xbe, 0xef}, Timestamp: ts}

	t.Run("json", func(t *testing.T) {
		b, err := encodeInner(trigger)
		require.NoError(t, err)
		require.True(t, raft.IsJSON(b))
		var got TriggerMessage
		require.NoError(t, decodeInner(b, &got))
		assert.Equal(t, trigger.LeaderID, got.LeaderID)
		assert.Equal(t, trigger.RoundID, got.RoundID)
		assert.Equal(t, ts.UnixNano(), got.Timestamp.UnixNano())
	})

	t.Run("msgpack", func(t *testing.T) {
		t.Setenv("P2P_ENCODING", "msgpack")

		b, err := encodeInner(trigger)
		require.NoError(t, err)
		require.False(t, raft.IsJSON(b))
		assert.False(t, bytes.Contains(b, []byte(pid)), "leader peer ID must be binary on the wire")
		var got TriggerMessage
		require.NoError(t, decodeInner(b, &got))
		assert.Equal(t, trigger.LeaderID, got.LeaderID, "binary peer ID must reconstruct base58 string")
		assert.Equal(t, trigger.RoundID, got.RoundID)
		assert.Equal(t, ts.UnixNano(), got.Timestamp.UnixNano(), "nanos timestamp must round-trip")

		pb, err := encodeInner(proof)
		require.NoError(t, err)
		var gotProof ProofMessage
		require.NoError(t, decodeInner(pb, &gotProof))
		assert.Equal(t, proof.RoundID, gotProof.RoundID)
		assert.Equal(t, proof.Value, gotProof.Value)
		assert.Equal(t, proof.Proof, gotProof.Proof)
		assert.Equal(t, ts.UnixNano(), gotProof.Timestamp.UnixNano())
	})

	// decode-both: JSON-configured node decodes a msgpack payload
	t.Run("decode_both", func(t *testing.T) {
		t.Setenv("P2P_ENCODING", "msgpack")
		mp, err := encodeInner(trigger)
		require.NoError(t, err)
		t.Setenv("P2P_ENCODING", "json")
		js, err := encodeInner(trigger)
		require.NoError(t, err)

		for name, data := range map[string][]byte{"json": js, "msgpack": mp} {
			var got TriggerMessage
			require.NoError(t, decodeInner(data, &got), name)
			assert.Equal(t, trigger.LeaderID, got.LeaderID, name)
			assert.Equal(t, trigger.RoundID, got.RoundID, name)
			assert.Equal(t, ts.UnixNano(), got.Timestamp.UnixNano(), name)
		}
	})
}
