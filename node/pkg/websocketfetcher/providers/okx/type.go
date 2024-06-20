package okx

// rate limits to 3 request / sec
const URL = "wss://ws.okx.com:8443/ws/v5/public"

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
