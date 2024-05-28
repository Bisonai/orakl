package binance

const (
	URL = "wss://stream.binance.com:443/ws"
)

type Stream string

type Subscription struct {
	Method string   `json:"method"`
	Params []Stream `json:"params"`
	Id     uint32   `json:"id"`
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
