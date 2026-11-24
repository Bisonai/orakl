package kraken

const URL = "wss://ws.kraken.com/v2"

type Params struct {
	Channel string   `json:"channel"`
	Symbol  []string `json:"symbol"`
	// EventTrigger "bbo" pushes the ticker on best bid/ask changes, not
	// only on trades.
	EventTrigger string `json:"event_trigger,omitempty"`
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
		Bid    float64 `json:"bid"`
		Ask    float64 `json:"ask"`
	} `json:"data"`
}
