package btse

const URL = "wss://ws.btse.com/ws/spot"

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
