package gemini

// top_of_book with bids/offers: "change" events carry the best bid/ask (one
// side per event), "trade" events the trades.
const URL = "wss://api.gemini.com/v1/multimarketdata?bids=true&heartbeat=true&offers=true&top_of_book=true&symbols="

// rate limit of http request: 120/min.
// recommended: 1/sec
// ex: https://api.gemini.com/v1/pubticker/btcusd
const TICKER_ENDPOINT = "https://api.gemini.com/v1/pubticker/"

type Response struct {
	Type        string  `json:"type"`
	TimestampMs *int64  `json:"timestampms"`
	Events      []Event `json:"events"`
}

type Event struct {
	Type      string  `json:"type"`
	Symbol    string  `json:"symbol"`
	Price     string  `json:"price"`
	Side      *string `json:"side"`
	Reason    *string `json:"reason"`
	Remaining *string `json:"remaining"`
	Delta     *string `json:"delta"`
	Amount    *string `json:"amount"`
	MakerSide *string `json:"makerSide"`
}

type HttpTickerResponse struct {
	Volume map[string]any `json:"volume"`
}
