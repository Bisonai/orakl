package bitmart

const URL = "wss://ws-manager-compress.bitmart.com/api?protocol=1.1"

type Subscription struct {
	Operation string   `json:"op"`
	Args      []string `json:"args"`
}

type Response struct {
	Table string `json:"table"`
	Data  []struct {
		Price  string `json:"last_price"`
		Symbol string `json:"symbol"`
		Volume string `json:"base_volume_24h"`
		Time   int64  `json:"ms_t"`
	} `json:"data"`
}
