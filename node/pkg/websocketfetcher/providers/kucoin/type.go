package kucoin

const (
	TokenUrl              = "https://api.kucoin.com/api/v1/bullet-public"
	URL                   = "wss://ws-api-spot.kucoin.com/"
	DEFAULT_PING_INTERVAL = 18000
)

type TokenResponse struct {
	Data ResponseData `json:"data"`
}

type InstanceServers struct {
	PingInterval int `json:"pingInterval"`
}

type ResponseData struct {
	Token           string            `json:"token"`
	InstanceServers []InstanceServers `json:"instanceServers"`
}

type Subscription struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Topic    string `json:"topic"`
	Response bool   `json:"response"`
}

type Ping struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type Data struct {
	Sequence    string `json:"sequence"`
	Price       string `json:"price"`
	Size        string `json:"size"`
	BestAsk     string `json:"bestAsk"`
	BestAskSize string `json:"bestAskSize"`
	BestBid     string `json:"bestBid"`
	BestBidSize string `json:"bestBidSize"`
	Time        int64  `json:"time"`
}

type Raw struct {
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Subject string `json:"subject"`
	Data    Data   `json:"data"`
}
