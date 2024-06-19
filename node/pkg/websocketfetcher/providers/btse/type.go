package btse

const URL = "wss://ws.btse.com/ws/spot"

// rate limit 15 request / sec
const MARKET_SUMMARY_ENDPOINT = "https://api.btse.com/spot/api/v3.2/market_summary"

type Subscription struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type Response struct {
	Topic string `json:"topic"`
	Data  []struct {
		Symbol    string  `json:"symbol"`
		Side      string  `json:"side"`
		Size      float64 `json:"size"`
		Price     float64 `json:"price"`
		TradeId   int     `json:"tradeId"`
		Timestamp int64   `json:"timestamp"`
	} `json:"data"`
}

type MarketSummary struct {
	Symbol string  `json:"symbol"`
	Size   float64 `json:"size"`
}
