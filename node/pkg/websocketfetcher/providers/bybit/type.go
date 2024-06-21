package bybit

const URL = "wss://stream.bybit.com/spot/public/v3"

type Subscription struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type Response struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Time   *int64  `json:"t"`
		Symbol *string `json:"s"`
		Price  *string `json:"c"`
		Volume *string `json:"v"`
	} `json:"data"`
}
