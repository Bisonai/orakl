package kraken

const URL = "wss://ws.kraken.com/v2"

type Params struct {
	Channel string   `json:"channel"`
	Symbol  []string `json:"symbol"`
}

type Subscription struct {
	Method string `json:"method"`
	Params Params `json:"params"`
}

type Response struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Data    []struct {
		Symbol string  `json:"symbol"`
		Price  float64 `json:"last"`
		Volume float64 `json:"volume"`
	} `json:"data"`
}
