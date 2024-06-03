package bithumb

const URL = "wss://pubwss.bithumb.com/pub/ws"

type Subscription struct {
	Type      string   `json:"type"`
	Symbols   []string `json:"symbols"`
	TickTypes []string `json:"tickTypes"`
}

type RawResponse struct {
	Type string `json:"type"`
}

type TransactionResponse struct {
	Type    string `json:"type"`
	Content struct {
		List []struct {
			Symbol    string `json:"symbol"`
			BuySellGb string `json:"buySellGb"`
			ContPrice string `json:"contPrice"`
			ContQty   string `json:"contQty"`
			ContAmt   string `json:"contAmt"`
			ContDtm   string `json:"contDtm"`
			Updn      string `json:"updn"`
		} `json:"list"`
	} `json:"content"`
}

type TickerResponse struct {
	Type    string `json:"type"`
	Content struct {
		Symbol         string `json:"symbol"`
		TickType       string `json:"tickType"`
		Date           string `json:"date"`
		Time           string `json:"time"`
		OpenPrice      string `json:"openPrice"`
		ClosePrice     string `json:"closePrice"`
		LowPrice       string `json:"lowPrice"`
		HighPrice      string `json:"highPrice"`
		Value          string `json:"value"`
		Volume         string `json:"volume"`
		SellVolume     string `json:"sellVolume"`
		BuyVolume      string `json:"buyVolume"`
		PrevClosePrice string `json:"prevClosePrice"`
		ChgRate        string `json:"chgRate"`
		ChgAmt         string `json:"chgAmt"`
		VolumePower    string `json:"volumePower"`
	} `json:"content"`
}
