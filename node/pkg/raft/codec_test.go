//nolint:all
package raft

import (
	"bytes"
	"testing"
	"time"

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

// TestCodecOuterRoundTripBothFormats proves decode-both: a Message survives a
// round-trip through the JSON codec and through the msgpack codec, and that the
// msgpack form carries a binary peer ID (not the base58 string) and millis.
func TestCodecOuterRoundTripBothFormats(t *testing.T) {
	pid := testPeerID(t)
	// millis-aligned so both formats compare exactly at millisecond granularity
	ts := time.UnixMilli(time.Now().UnixMilli())

	base := Message{
		Type:      Heartbeat,
		SentFrom:  pid,
		Data:      []byte(`{"leaderID":"x","term":1}`),
		Timestamp: ts,
	}

	// JSON path (P2P_ENCODING unset / default)
	t.Run("json", func(t *testing.T) {
		jsonData, err := encodeMessage(base)
		require.NoError(t, err)
		require.True(t, IsJSON(jsonData), "legacy encode must be JSON by default")

		got, err := decodeMessage(jsonData)
		require.NoError(t, err)
		assert.Equal(t, base.Type, got.Type)
		assert.Equal(t, base.SentFrom, got.SentFrom)
		assert.Equal(t, ts.UnixMilli(), got.Timestamp.UnixMilli())
		assert.Equal(t, []byte(base.Data), []byte(got.Data))
	})

	// msgpack path (P2P_ENCODING=msgpack)
	t.Run("msgpack", func(t *testing.T) {
		t.Setenv("P2P_ENCODING", "msgpack")
		mpData, err := encodeMessage(base)
		require.NoError(t, err)
		require.False(t, IsJSON(mpData), "msgpack encode must not look like JSON")
		assert.False(t, bytes.Contains(mpData, []byte(pid)), "peer ID must be binary on the wire, not base58")
		assert.Less(t, len(mpData), len(mustJSON(t, base)), "msgpack should be smaller than JSON")

		got, err := decodeMessage(mpData)
		require.NoError(t, err)
		assert.Equal(t, base.Type, got.Type, "type enum must round-trip")
		assert.Equal(t, base.SentFrom, got.SentFrom, "binary peer ID must reconstruct base58 string")
		assert.Equal(t, ts.UnixMilli(), got.Timestamp.UnixMilli(), "millis timestamp must round-trip")
		assert.Equal(t, []byte(base.Data), []byte(got.Data))
	})
}

// TestCodecDecodeBoth proves a node running the default JSON encoder can still
// decode a message that was produced by the msgpack encoder, and vice versa.
func TestCodecDecodeBoth(t *testing.T) {
	pid := testPeerID(t)
	ts := time.UnixMilli(time.Now().UnixMilli())
	base := Message{Type: RequestVote, SentFrom: pid, Data: []byte(`{"term":7}`), Timestamp: ts}

	// produce msgpack
	t.Setenv("P2P_ENCODING", "msgpack")
	mpData, err := encodeMessage(base)
	require.NoError(t, err)

	// produce JSON
	t.Setenv("P2P_ENCODING", "json")
	jsonData, err := encodeMessage(base)
	require.NoError(t, err)

	// a JSON-configured node (current env) decodes both
	for name, data := range map[string][]byte{"json": jsonData, "msgpack": mpData} {
		got, err := decodeMessage(data)
		require.NoError(t, err, name)
		assert.Equal(t, base.Type, got.Type, name)
		assert.Equal(t, base.SentFrom, got.SentFrom, name)
		assert.Equal(t, ts.UnixMilli(), got.Timestamp.UnixMilli(), name)
	}
}

func TestCodecInnerRoundTrip(t *testing.T) {
	pid := testPeerID(t)

	t.Run("json", func(t *testing.T) {
		hb := HeartbeatMessage{LeaderID: pid, Term: 3}
		b, err := encodeInner(hb)
		require.NoError(t, err)
		require.True(t, IsJSON(b))
		var got HeartbeatMessage
		require.NoError(t, decodeInner(b, &got))
		assert.Equal(t, hb, got)
	})

	t.Run("msgpack", func(t *testing.T) {
		t.Setenv("P2P_ENCODING", "msgpack")
		hb := HeartbeatMessage{LeaderID: pid, Term: 3}
		b, err := encodeInner(hb)
		require.NoError(t, err)
		require.False(t, IsJSON(b))
		assert.False(t, bytes.Contains(b, []byte(pid)), "inner LeaderID must be binary on the wire")
		var got HeartbeatMessage
		require.NoError(t, decodeInner(b, &got))
		assert.Equal(t, hb, got)

		rv := ReplyRequestVoteMessage{VoteGranted: true, LeaderID: pid}
		rb, err := encodeInner(rv)
		require.NoError(t, err)
		var gotRV ReplyRequestVoteMessage
		require.NoError(t, decodeInner(rb, &gotRV))
		assert.Equal(t, rv, gotRV)
	})
}

func mustJSON(t *testing.T, msg Message) []byte {
	t.Helper()
	// force JSON regardless of ambient env
	t.Setenv("P2P_ENCODING", "json")
	b, err := encodeMessage(msg)
	require.NoError(t, err)
	t.Setenv("P2P_ENCODING", "msgpack")
	return b
}
