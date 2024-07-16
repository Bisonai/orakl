package gopax

import "encoding/json"

const URL = "wss://wsapi.gopax.co.kr"

type Subscription struct {
	Name   string      `json:"n"`
	Object interface{} `json:"o"`
}

type Response struct {
	Type   int             `json:"i"`
	Name   string          `json:"n"`
	Object json.RawMessage `json:"o"`
}

type InitialResponse struct {
	Data []Ticker `json:"data"`
}

type Tickers map[string]Ticker

type Ticker struct {
	Volume    float64 `json:"baseVolume"`
	Timestamp int64   `json:"lastTraded"`
	Price     float64 `json:"last"`
	Name      string  `json:"tradingPairName"`
}
