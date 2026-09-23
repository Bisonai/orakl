package raft

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/vmihailenco/msgpack/v5"
)

// P2P wire codec.
//
// Two-phase rollout: decode always accepts BOTH legacy JSON and new msgpack.
// encode chooses the format based on the P2P_ENCODING env var (default json),
// so a Phase-1 deploy runs with P2P_ENCODING unset (still emitting JSON) while
// every node already understands msgpack. Phase-2 flips P2P_ENCODING=msgpack
// once all nodes are upgraded.
//
// msgpack wire format additionally shrinks per-message bytes by:
//   - encoding peer IDs as raw bytes instead of 52-char base58 strings
//   - encoding timestamps as int64 unix-nanos instead of RFC3339 strings
//     (unix-nanos is the same 9 bytes as unix-millis in msgpack but keeps the
//     full nanosecond precision the JSON path already carries)
//   - encoding message Type as a small int enum instead of the string constant
//
// The in-memory Go types (Message, HeartbeatMessage, ...) keep string peer IDs
// and time.Time timestamps; all conversions live here in the codec layer.

// message type int enum used on the msgpack wire. Aggregator message types are
// referenced by their string value to avoid an import cycle (aggregator imports
// raft, not the other way around).
var typeToInt = map[MessageType]int{
	Heartbeat:                1,
	RequestVote:              2,
	ReplyVote:                3,
	AppendEntries:            4,
	ReplyAppendEntries:       5,
	MessageType("trigger"):   6,
	MessageType("priceData"): 7,
	MessageType("priceFix"):  8,
	MessageType("proof"):     9,
}

var intToType = func() map[int]MessageType {
	m := make(map[int]MessageType, len(typeToInt))
	for k, v := range typeToInt {
		m[v] = k
	}
	return m
}()

// UseMsgpack reports whether new messages should be encoded as msgpack.
// Controlled by the P2P_ENCODING env var (values: "json" (default) | "msgpack").
func UseMsgpack() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("P2P_ENCODING")), "msgpack")
}

// IsJSON sniffs the first non-space byte: a legacy JSON object starts with '{'.
// msgpack-encoded structs start with a fixmap/map marker (never 0x7b), so this
// cleanly distinguishes the two wire formats.
func IsJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		}
		return b == '{'
	}
	return false
}

// PeerIDToBytes converts a base58 peer ID string to its raw byte form.
func PeerIDToBytes(s string) ([]byte, error) {
	id, err := peer.Decode(s)
	if err != nil {
		return nil, err
	}
	return []byte(id), nil
}

// PeerIDFromBytes reconstructs the peer ID string from its raw byte form.
func PeerIDFromBytes(b []byte) (string, error) {
	id, err := peer.IDFromBytes(b)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// wireMessage is the compact msgpack representation of the outer Message.
type wireMessage struct {
	Type      int    `msgpack:"t"`
	SentFrom  []byte `msgpack:"f"`
	Data      []byte `msgpack:"d"`
	Timestamp int64  `msgpack:"ts"`
}

// encodeMessage marshals the outer Message using the configured wire format.
func encodeMessage(msg Message) ([]byte, error) {
	if !UseMsgpack() {
		return json.Marshal(msg)
	}

	from, err := PeerIDToBytes(msg.SentFrom)
	if err != nil {
		return nil, err
	}
	t, ok := typeToInt[msg.Type]
	if !ok {
		return nil, fmt.Errorf("raft: unknown message type %q", msg.Type)
	}
	return msgpack.Marshal(&wireMessage{
		Type:      t,
		SentFrom:  from,
		Data:      msg.Data,
		Timestamp: msg.Timestamp.UnixNano(),
	})
}

// decodeMessage unmarshals the outer Message, accepting both legacy JSON and
// msgpack (decode-both), preserving exact field semantics for the JSON path.
func decodeMessage(data []byte) (Message, error) {
	if IsJSON(data) {
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			return Message{}, err
		}
		return m, nil
	}

	var wm wireMessage
	if err := msgpack.Unmarshal(data, &wm); err != nil {
		return Message{}, err
	}
	from, err := PeerIDFromBytes(wm.SentFrom)
	if err != nil {
		return Message{}, err
	}
	t, ok := intToType[wm.Type]
	if !ok {
		return Message{}, fmt.Errorf("raft: unknown message type enum %d", wm.Type)
	}
	return Message{
		Type:      t,
		SentFrom:  from,
		Data:      wm.Data,
		Timestamp: time.Unix(0, wm.Timestamp),
	}, nil
}

// wire structs for the raft inner payloads.
type wireHeartbeat struct {
	LeaderID []byte `msgpack:"l"`
	Term     int    `msgpack:"tm"`
}

type wireRequestVote struct {
	Term int `msgpack:"tm"`
}

type wireReplyRequestVote struct {
	VoteGranted bool   `msgpack:"v"`
	LeaderID    []byte `msgpack:"l"`
}

// encodeInner marshals a raft inner payload using the configured wire format.
func encodeInner(v any) ([]byte, error) {
	if !UseMsgpack() {
		return json.Marshal(v)
	}

	switch m := v.(type) {
	case HeartbeatMessage:
		leader, err := PeerIDToBytes(m.LeaderID)
		if err != nil {
			return nil, err
		}
		return msgpack.Marshal(&wireHeartbeat{LeaderID: leader, Term: m.Term})
	case RequestVoteMessage:
		return msgpack.Marshal(&wireRequestVote{Term: m.Term})
	case ReplyRequestVoteMessage:
		leader, err := PeerIDToBytes(m.LeaderID)
		if err != nil {
			return nil, err
		}
		return msgpack.Marshal(&wireReplyRequestVote{VoteGranted: m.VoteGranted, LeaderID: leader})
	default:
		return nil, fmt.Errorf("raft: unknown inner message type %T", v)
	}
}

// decodeInner unmarshals a raft inner payload into v (a pointer), accepting both
// legacy JSON and msgpack.
func decodeInner(data []byte, v any) error {
	if IsJSON(data) {
		return json.Unmarshal(data, v)
	}

	switch m := v.(type) {
	case *HeartbeatMessage:
		var w wireHeartbeat
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		leader, err := PeerIDFromBytes(w.LeaderID)
		if err != nil {
			return err
		}
		m.LeaderID = leader
		m.Term = w.Term
		return nil
	case *RequestVoteMessage:
		var w wireRequestVote
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		m.Term = w.Term
		return nil
	case *ReplyRequestVoteMessage:
		var w wireReplyRequestVote
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		leader, err := PeerIDFromBytes(w.LeaderID)
		if err != nil {
			return err
		}
		m.VoteGranted = w.VoteGranted
		m.LeaderID = leader
		return nil
	default:
		return fmt.Errorf("raft: unknown inner message type %T", v)
	}
}
