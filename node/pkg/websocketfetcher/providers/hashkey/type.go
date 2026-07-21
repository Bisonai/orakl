package hashkey

const (
	// HashKey Global (not HashKey Pro at stream-pro.hashkey.com) public spot
	// market-data websocket, v2. v1 does not stream and the Pro host rejects the
	// pairs we care about with "Parameter error".
	URL = "wss://stream-glb.hashkey.com/quote/ws/v2"

	// The server never pings us; if the client stays silent it drops the
	// connection after ~30s. Sending a client ping every 10s (the interval the
	// docs specify) keeps it alive -- the server answers each with a pong.
	PingInterval = 10000
)

// Subscription is one `sub` request. HashKey takes a single symbol per request,
// so we send one per feed. Symbol format is the concatenated "<base><quote>"
// upper-case, e.g. GRNDUSDT -- the same key the Combined feed map uses.
type Subscription struct {
	Topic  string            `json:"topic"`
	Event  string            `json:"event"`
	Params map[string]string `json:"params"`
}

// Ping is the client keep-alive; the value is a millisecond timestamp.
type Ping struct {
	Ping int64 `json:"ping"`
}

// Response is a `realtimes` (24h ticker) push.
//
//	{"topic":"realtimes","params":{"symbol":"GRNDUSDT"},
//	 "data":{"t":..,"s":"GRNDUSDT","o":..,"h":..,"l":..,"c":"0.01016",
//	         "v":"1005953.72","qv":"10094.22","m":".."}}
type Response struct {
	Topic string     `json:"topic"`
	Data  TickerData `json:"data"`
}

type TickerData struct {
	Timestamp int64  `json:"t"`
	Symbol    string `json:"s"`
	Open      string `json:"o"`
	High      string `json:"h"`
	Low       string `json:"l"`
	Price     string `json:"c"`  // close / current price
	Volume    string `json:"v"`  // BASE-asset volume (coin amount) -- the one we want
	QuoteVol  string `json:"qv"` // quote-asset volume (USDT turnover)
}
