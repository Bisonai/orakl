package common

import (
	"fmt"
	"strings"
	"time"
)

// SimpleOrderbookType is the "ty" of orderbook messages in the SIMPLE format
// of upbit's websocket (also served by bithumb's upbit-compatible websocket).
const SimpleOrderbookType = "orderbook"

// SimpleOrderbookUnit is one price level of a SIMPLE-format orderbook.
type SimpleOrderbookUnit struct {
	AskPrice float64 `json:"ap"`
	BidPrice float64 `json:"bp"`
	AskSize  float64 `json:"as"`
	BidSize  float64 `json:"bs"`
}

// SimpleOrderbook is a SIMPLE-format orderbook message; units are sorted best
// first. Code is "<QUOTE>-<BASE>".
type SimpleOrderbook struct {
	Type      string                `json:"ty"`
	Code      string                `json:"cd"`
	Timestamp int64                 `json:"tms"`
	Units     []SimpleOrderbookUnit `json:"obu"`
}

// TopOfBook returns the feed IDs of the book's market (feedMap keyed
// "<BASE>-<QUOTE>") and its best bid/ask and timestamp (receive time if the
// message has none).
func (b SimpleOrderbook) TopOfBook(feedMap map[string][]int32) ([]int32, float64, float64, time.Time, error) {
	splitted := strings.Split(b.Code, "-")
	if len(splitted) != 2 {
		return nil, 0, 0, time.Time{}, fmt.Errorf("unexpected orderbook code %q", b.Code)
	}
	ids, exists := feedMap[strings.ToUpper(splitted[1])+"-"+strings.ToUpper(splitted[0])]
	if !exists {
		return nil, 0, 0, time.Time{}, fmt.Errorf("feed not found for orderbook %s", b.Code)
	}
	if len(b.Units) == 0 {
		return nil, 0, 0, time.Time{}, fmt.Errorf("empty orderbook for %s", b.Code)
	}
	ts := time.Now()
	if b.Timestamp > 0 {
		ts = unixEpoch(b.Timestamp)
	}
	return ids, b.Units[0].BidPrice, b.Units[0].AskPrice, ts, nil
}

// unixEpoch reads an epoch timestamp in milli-, micro- or nanoseconds by its
// magnitude: upbit sends "tms" in ms, bithumb's orderbook in µs.
func unixEpoch(v int64) time.Time {
	switch {
	case v >= 1e17:
		return time.Unix(0, v)
	case v >= 1e14:
		return time.UnixMicro(v)
	default:
		return time.UnixMilli(v)
	}
}
