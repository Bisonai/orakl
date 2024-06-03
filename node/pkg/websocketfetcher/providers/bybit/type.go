package bybit

const URL = "wss://stream.bybit.com/contract/usdt/public/v3"

type Subscription struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type Response struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Symbol            *string `json:"symbol"`
		TickDirection     *string `json:"tickDirection"`
		Price24hPcnt      *string `json:"price24hPcnt"`
		LastPrice         *string `json:"lastPrice"`
		PrevPrice24h      *string `json:"prevPrice24h"`
		HighPrice24h      *string `json:"highPrice24h"`
		LowPrice24h       *string `json:"lowPrice24h"`
		PrevPrice1h       *string `json:"prevPrice1h"`
		MarkPrice         *string `json:"markPrice"`
		IndexPrice        *string `json:"indexPrice"`
		OpenInterest      *string `json:"openInterest"`
		OpenInterestValue *string `json:"openInterestValue"`
		Turnover24h       *string `json:"turnover24h"`
		Volume24h         *string `json:"volume24h"`
		NextFundingTime   *string `json:"nextFundingTime"`
		FundingRate       *string `json:"fundingRate"`
		Bid1Price         *string `json:"bid1Price"`
		Bid1Size          *string `json:"bid1Size"`
		Ask1Price         *string `json:"ask1Price"`
		Ask1Size          *string `json:"ask1Size"`
	} `json:"data"`
	Cs int64 `json:"cs"`
	Ts int64 `json:"ts"`
}
