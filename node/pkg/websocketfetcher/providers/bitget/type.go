package bitget

const URL = "wss://ws.bitget.com/v2/ws/public"

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
	Data   []Data
}

type Data struct {
	InstId    string `json:"instId"`
	Price     string `json:"lastPR"`
	Volume    string `json:"baseVolume"`
	Timestamp string `json:"ts"`
}
