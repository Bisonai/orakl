package coinone

const (
	URL = "wss://stream.coinone.co.kr"
)

type Topic struct {
	QuoteCurrency  string `json:"quote_currency"`
	TargetCurrency string `json:"target_currency"`
}

type Subscription struct {
	RequestType string `json:"request_type"`
	Channel     string `json:"channel"`
	Topic       Topic  `json:"topic"`
	Format      string `json:"format"`
}

type Data struct {
	QuoteCurrency         string `json:"qc"`
	TargetCurrency        string `json:"tc"`
	Timestamp             int64  `json:"t"`
	QuoteVolume           string `json:"qv"`
	TargetVolume          string `json:"tv"`
	First                 string `json:"fi"`
	Low                   string `json:"lo"`
	High                  string `json:"hi"`
	Last                  string `json:"la"`
	VolumePower           string `json:"vp"`
	AskBestPrice          string `json:"abp"`
	AskBestQty            string `json:"abq"`
	BidBestPrice          string `json:"bbp"`
	BidBestQty            string `json:"bbq"`
	ID                    string `json:"i"`
	YesterdayFirst        string `json:"yfi"`
	YesterdayLow          string `json:"ylo"`
	YesterdayHigh         string `json:"yhi"`
	YesterdayLast         string `json:"yla"`
	YesterdayQuoteVolume  string `json:"yqv"`
	YesterdayTargetVolume string `json:"ytv"`
}

type Raw struct {
	ResponseType string `json:"r"`
	Channel      string `json:"c"`
	Data         Data   `json:"d"`
}
