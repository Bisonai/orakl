package bybit

const URL = "wss://stream.bybit.com/v5/public/spot"

type Subscription struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}
type Response struct {
	Topic     *string `json:"topic"`
	Timestamp *int64  `json:"ts"`
	Type      *string `json:"type"`
	Data      struct {
		Symbol string `json:"symbol"`
		Price  string `json:"lastPrice"`
		Volume string `json:"volume24h"`
	}
}

type Heartbeat struct {
	Op string `json:"op"`
}
