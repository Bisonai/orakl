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

type MarketSnapshotSingleData struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"lastTradedPrice"`
	Volume float64 `json:"vol"`
	Time   int64   `json:"datetime"`
}

type MarketSnapshotData struct {
	Sequence string                     `json:"sequence"`
	Data     []MarketSnapshotSingleData `json:"data"`
}

type MarketSnapshotRaw struct {
	Type    string              `json:"type"`
	Topic   string              `json:"topic"`
	Subject string              `json:"subject"`
	Data    *MarketSnapshotData `json:"data"`
}
