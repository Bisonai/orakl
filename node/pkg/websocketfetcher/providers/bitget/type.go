package bitget

// TODO: migrate to v2 (reference: https://www.bitget.com/api-doc/spot/websocket/public/Tickers-Channel)
// wss://ws.bitget.com/v2/ws/public
const URL = "wss://ws.bitget.com/mix/v1/stream"

type Arg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstId   string `json:"instId"`
}

type Subscription struct {
	Op   string `json:"op"`
	Args []Arg  `json:"args"`
}

type Response struct {
	Action *string `json:"action"`
	Arg    Arg     `json:"arg"`
	Data   []Data  `json:"data"`
}

type Data struct {
	InstId      string `json:"instId"`
	Last        string `json:"last"`
	BestAsk     string `json:"bestAsk"`
	BestBid     string `json:"bestBid"`
	Open24h     string `json:"open24h"`
	High24h     string `json:"high24h"`
	Low24h      string `json:"low24h"`
	BaseVolume  string `json:"baseVolume"`
	QuoteVolume string `json:"quoteVolume"`
	Ts          int64  `json:"ts"`
	LabelId     int64  `json:"labelId"`
	OpenUtc     string `json:"openUtc"`
	ChgUTC      string `json:"chgUTC"`
	BidSz       string `json:"bidSz"`
	AskSz       string `json:"askSz"`
}
