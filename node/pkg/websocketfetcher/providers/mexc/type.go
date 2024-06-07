package mexc

const URL = "wss://wbs.mexc.com/ws"

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Ticker struct {
	Symbol                string `json:"s"`
	Price                 string `json:"p"`
	PercentChange         string `json:"r"`
	TimezonePercentChange string `json:"tr"`
	HighPrice             string `json:"h"`
	LowPrice              string `json:"l"`
	Volume                string `json:"v"`
	QuoteVolume           string `json:"q"`
	LastRT                string `json:"lastRT"`
	MergeTimes            string `json:"MT"`
	NetValue              string `json:"NV"`
	Time                  string `json:"t"`
}

type Response struct {
	Channel string `json:"c"`
	Data    Ticker `json:"d"`
}

type BatchResponse struct {
	Channel string   `json:"c"`
	Data    []Ticker `json:"d"`
}
