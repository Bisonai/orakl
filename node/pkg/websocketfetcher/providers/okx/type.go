package okx

// rate limits to 3 request / sec
const URL = "wss://ws.okx.com/ws/v5/public"

type Arg struct {
	Channel string `json:"channel"`
	InstId  string `json:"instId"`
}

type Subscription struct {
	Operation string `json:"op"`
	Args      []Arg  `json:"args"`
}

type Response struct {
	Event *string `json:"event"`
	Arg   Arg     `json:"arg"`
	Data  []struct {
		InstId    string `json:"instId"`
		Price     string `json:"last"`
		Volume    string `json:"vol24h"`
		Timestamp string `json:"ts"`
	} `json:"data"`
}

// BboChannel is the tick-by-tick best bid/ask channel.
const BboChannel = "bbo-tbt"

// BboResponse is a bbo-tbt push: one level per side, ts in ms.
type BboResponse struct {
	Arg  Arg `json:"arg"`
	Data []struct {
		Asks      [][]string `json:"asks"`
		Bids      [][]string `json:"bids"`
		Timestamp string     `json:"ts"`
	} `json:"data"`
}
