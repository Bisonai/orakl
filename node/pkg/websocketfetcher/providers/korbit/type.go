package korbit

const (
	URL = "wss://ws-api.korbit.co.kr/v2/public"
)

type Raw struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Symbol    string `json:"symbol"`
	Snapshot  bool   `json:"snapshot"`
	Data      Ticker `json:"data"`
}

type Ticker struct {
	Open               string `json:"open"`
	High               string `json:"high"`
	Low                string `json:"low"`
	Close              string `json:"close"`
	PrevClose          string `json:"prevClose"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	BestAskPrice       string `json:"bestAskPrice"`
	BestBidPrice       string `json:"bestBidPrice"`
	LastTradedAt       int64  `json:"lastTradedAt"`
}

type Subscription struct {
	Method  string   `json:"method"`
	Type    string   `json:"type"`
	Symbols []string `json:"symbols"`
}
