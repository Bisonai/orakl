package aggregator

import (
	"encoding/json"
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/raft"
	"github.com/vmihailenco/msgpack/v5"
)

// P2P wire codec for aggregator inner payloads.
//
// Consistent with the raft outer codec: encode chooses the wire format from the
// P2P_ENCODING env var (default json), decode accepts BOTH legacy JSON and
// msgpack. msgpack shrinks bytes by encoding timestamps as int64 unix-millis and
// peer IDs as raw bytes. The in-memory structs keep time.Time / string peer IDs;
// conversions live here.

type wireTrigger struct {
	LeaderID  []byte `msgpack:"l"`
	RoundID   int32  `msgpack:"r"`
	Timestamp int64  `msgpack:"ts"`
}

type wirePriceData struct {
	RoundID   int32 `msgpack:"r"`
	PriceData int64 `msgpack:"p"`
	Timestamp int64 `msgpack:"ts"`
}

type wirePriceFix struct {
	RoundID   int32 `msgpack:"r"`
	PriceData int64 `msgpack:"p"`
	Timestamp int64 `msgpack:"ts"`
}

type wireProof struct {
	RoundID   int32  `msgpack:"r"`
	Value     int64  `msgpack:"v"`
	Proof     []byte `msgpack:"pf"`
	Timestamp int64  `msgpack:"ts"`
}

// encodeInner marshals an aggregator inner payload using the configured format.
func encodeInner(v any) ([]byte, error) {
	if !raft.UseMsgpack() {
		return json.Marshal(v)
	}

	switch m := v.(type) {
	case TriggerMessage:
		leader, err := raft.PeerIDToBytes(m.LeaderID)
		if err != nil {
			return nil, err
		}
		return msgpack.Marshal(&wireTrigger{
			LeaderID:  leader,
			RoundID:   m.RoundID,
			Timestamp: m.Timestamp.UnixMilli(),
		})
	case PriceDataMessage:
		return msgpack.Marshal(&wirePriceData{
			RoundID:   m.RoundID,
			PriceData: m.PriceData,
			Timestamp: m.Timestamp.UnixMilli(),
		})
	case PriceFixMessage:
		return msgpack.Marshal(&wirePriceFix{
			RoundID:   m.RoundID,
			PriceData: m.PriceData,
			Timestamp: m.Timestamp.UnixMilli(),
		})
	case ProofMessage:
		return msgpack.Marshal(&wireProof{
			RoundID:   m.RoundID,
			Value:     m.Value,
			Proof:     m.Proof,
			Timestamp: m.Timestamp.UnixMilli(),
		})
	default:
		return nil, fmt.Errorf("aggregator: unknown inner message type %T", v)
	}
}

// decodeInner unmarshals an aggregator inner payload into v (a pointer),
// accepting both legacy JSON and msgpack.
func decodeInner(data []byte, v any) error {
	if raft.IsJSON(data) {
		return json.Unmarshal(data, v)
	}

	switch m := v.(type) {
	case *TriggerMessage:
		var w wireTrigger
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		leader, err := raft.PeerIDFromBytes(w.LeaderID)
		if err != nil {
			return err
		}
		m.LeaderID = leader
		m.RoundID = w.RoundID
		m.Timestamp = time.UnixMilli(w.Timestamp)
		return nil
	case *PriceDataMessage:
		var w wirePriceData
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		m.RoundID = w.RoundID
		m.PriceData = w.PriceData
		m.Timestamp = time.UnixMilli(w.Timestamp)
		return nil
	case *PriceFixMessage:
		var w wirePriceFix
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		m.RoundID = w.RoundID
		m.PriceData = w.PriceData
		m.Timestamp = time.UnixMilli(w.Timestamp)
		return nil
	case *ProofMessage:
		var w wireProof
		if err := msgpack.Unmarshal(data, &w); err != nil {
			return err
		}
		m.RoundID = w.RoundID
		m.Value = w.Value
		m.Proof = w.Proof
		m.Timestamp = time.UnixMilli(w.Timestamp)
		return nil
	default:
		return fmt.Errorf("aggregator: unknown inner message type %T", v)
	}
}
