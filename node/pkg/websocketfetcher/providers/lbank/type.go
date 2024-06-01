package lbank

const URL = "wss://www.lbkex.net/ws/V2/"

type Subscription struct {
	Action    string `json:"action"`
	Subscribe string `json:"subscribe"`
	Pair      string `json:"pair"`
}

type Response struct {
	Tick struct {
		ToCny    float64 `json:"to_cny"`
		High     float64 `json:"high"`
		Vol      float64 `json:"vol"`
		Low      float64 `json:"low"`
		Change   float64 `json:"change"`
		Usd      float64 `json:"usd"`
		ToUsd    float64 `json:"to_usd"`
		Dir      string  `json:"dir"`
		Turnover float64 `json:"turnover"`
		Latest   float64 `json:"latest"`
		Cny      float64 `json:"cny"`
	} `json:"tick"`
	Type   string `json:"type"`
	Pair   string `json:"pair"`
	Server string `json:"SERVER"`
	TS     string `json:"TS"`
}
