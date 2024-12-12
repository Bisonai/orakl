package coinex

const URL = "wss://socket.coinex.com/v2/spot"

type SubscribeParams struct {
	MarketList []string `json:"market_list"`
}

type Subscription struct {
	Method string          `json:"method"`
	Params SubscribeParams `json:"params"`
	ID     int             `json:"id"`
}

type State struct {
	Market     string `json:"market"`
	Last       string `json:"last"`
	Open       string `json:"open"`
	Close      string `json:"close"`
	High       string `json:"high"`
	Low        string `json:"low"`
	Volume     string `json:"volume"`
	VolumeSell string `json:"volume_sell"`
	VolumeBuy  string `json:"volume_buy"`
	Value      string `json:"value"`
	Period     int    `json:"period"`
}

type Data struct {
	StateList []State `json:"state_list"`
}

type Response struct {
	Method string `json:"method"`
	Data   Data   `json:"data"`
	ID     *int   `json:"id"`
}
