package coinex

const URL = "wss://socket.coinex.com/"

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

type Ticker struct {
	Open      string `json:"open"`
	Last      string `json:"last"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Deal      string `json:"deal"`
	Volume    string `json:"volume"`
	SellTotal string `json:"sell_total"`
	BuyTotal  string `json:"buy_total"`
	Period    int    `json:"period"`
}

type Response struct {
	Method string              `json:"method"`
	Params []map[string]Ticker `json:"params"`
	ID     *int                `json:"id"`
}
