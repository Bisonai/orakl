package mexc

const (
	// MEXC retired the JSON push channels. Subscribing to the old
	// `spot@public.miniTickers.v3.api@UTC+0` topic on wss://wbs.mexc.com/ws is
	// still accepted at the TCP level but answered with
	//   {"id":0,"code":0,"msg":"Not Subscribed successfully! [...] Reason： Blocked! "}
	// after which the connection stays open and completely silent -- which is why
	// this went unnoticed until every mexc feed had been stale for weeks.
	//
	// The protobuf channels on wbs-api replace them. The `.pb` suffix on the topic
	// is what selects them, and they push binary frames (see readFrame).
	URL   = "wss://wbs-api.mexc.com/ws"
	Topic = "spot@public.miniTickers.v3.api.pb@UTC+0"

	// miniTickers pushes the whole spot market, batched by MEXC into frames whose
	// size we do not control. nhooyr's 32KB default read limit would fail the read
	// on an oversized frame and drop the connection.
	ReadLimit = 4 * 1024 * 1024
)

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// Ack is the text frame MEXC answers a SUBSCRIPTION request with. On success Msg
// echoes the subscribed topic back.
type Ack struct {
	ID   int    `json:"id"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
