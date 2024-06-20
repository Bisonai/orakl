package bingx

// rate limit 100 req / 10 sec
const URL = "wss://open-api-ws.bingx.com/market"

type Subscription struct {
	ID          string `json:"id"`
	RequestType string `json:"reqType"`
	DataType    string `json:"dataType"`
}

type Response struct {
	Code      int   `json:"code"`
	Data      Data  `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

type Data struct {
	EventType string  `json:"e"`
	EventTime int64   `json:"E"`
	Symbol    string  `json:"s"`
	Price     float64 `json:"c"`
	Volume    float64 `json:"v"`
}

type Heartbeat struct {
	Ping string `json:"ping"`
	Time string `json:"time"`
}

type HeartbeatResonse struct {
	Pong string `json:"pong"`
	Time string `json:"time"`
}
