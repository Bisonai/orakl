package binance

const (
	// Combined-stream endpoint: every message is {"stream": ..., "data": ...},
	// which names the symbol of partial book depth messages (their payload
	// doesn't).
	URL = "wss://stream.binance.com:443/stream"

	MiniTickerSuffix = "@miniTicker"
	// Top 5 levels, pushed at most once a second (only when changed): the
	// tick-by-tick @bookTicker pushes ~170 messages/s for BTCUSDT alone.
	DepthSuffix = "@depth5@1000ms"
)

type Stream string

type Subscription struct {
	Method string   `json:"method"`
	Params []Stream `json:"params"`
	ID     uint32   `json:"id"`
}

type MiniTicker struct {
	EventType   string `json:"e"`
	EventTime   int64  `json:"E"`
	Symbol      string `json:"s"`
	Price       string `json:"c"`
	OpenPrice   string `json:"o"`
	HighPrice   string `json:"h"`
	LowPrice    string `json:"l"`
	Volume      string `json:"v"`
	QuoteVolume string `json:"q"`
}

// CombinedMessage is the envelope of the combined-stream endpoint.
type CombinedMessage struct {
	Stream string         `json:"stream"`
	Data   map[string]any `json:"data"`
}

// PartialDepth is a <symbol>@depth<levels>@<speed> message.
type PartialDepth struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}
