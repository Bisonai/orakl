package korbit

const (
	URL = "wss://ws2.korbit.co.kr/v1/user/push"
)

type Raw struct {
	Event string `json:"event"`
	Data  Ticker `json:"data"`
}

type Ticker struct {
	Channel      string `json:"channel"`
	CurrencyPair string `json:"currency_pair"`
	Timestamp    int64  `json:"timestamp"`
	Last         string `json:"last"`
	Open         string `json:"open"`
	Bid          string `json:"bid"`
	Ask          string `json:"ask"`
	Low          string `json:"low"`
	High         string `json:"high"`
	Volume       string `json:"volume"`
	Change       string `json:"change"`
}

type Subscription struct {
	AccessToken *string `json:"accessToken"`
	Timestamp   int64   `json:"timestamp"`
	Event       string  `json:"event"`
	Data        Data    `json:"data"`
}

type Data struct {
	Channels []string `json:"channels"`
}
