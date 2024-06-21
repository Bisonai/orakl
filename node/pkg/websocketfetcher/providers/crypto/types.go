package crypto

const URL = "wss://stream.crypto.com/v2/market"

type Subscription struct {
	Method string `json:"method"`
	Params struct {
		Channels []string `json:"channels"`
	} `json:"params"`
}

type Response struct {
	Method string `json:"method"`
	Code   int    `json:"code"`
	Result struct {
		Channel        string `json:"channel"`
		InstrumentName string `json:"instrument_name"`
		Subscription   string `json:"subscription"`
		Data           []struct {
			HighPrice24h        string  `json:"h"`
			LowPrice24h         *string `json:"l"`
			LastTradePrice      *string `json:"a"`
			InstrumentName      string  `json:"i"`
			Total24hTradeVolume string  `json:"v"`
			Total24hTradeValue  string  `json:"vv"`
			OpenInterest        string  `json:"oi"`
			PriceChange24h      *string `json:"c"`
			CurrentBestBidPrice *string `json:"b"`
			CurrentBestAskPrice *string `json:"k"`
			Timestamp           int64   `json:"t"`
		} `json:"data"`
	} `json:"result"`
}

type Heartbeat struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
}
