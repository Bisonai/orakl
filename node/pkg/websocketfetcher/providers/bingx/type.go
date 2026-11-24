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
	// Best ask/bid price and size. Both cases are declared: encoding/json
	// matches keys case-insensitively when no field has the exact name, so
	// with only "A"/"B" the sizes "a"/"b" would overwrite the prices.
	AskPrice float64 `json:"A"`
	AskSize  float64 `json:"a"`
	BidPrice float64 `json:"B"`
	BidSize  float64 `json:"b"`
}

type Heartbeat struct {
	Ping string `json:"ping"`
	Time string `json:"time"`
}

type HeartbeatResonse struct {
	Pong string `json:"pong"`
	Time string `json:"time"`
}
