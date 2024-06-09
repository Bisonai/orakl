package upbit

const URL = "wss://api.upbit.com/websocket/v1"

type Subscription []interface{}

type Response struct {
	Type               string   `json:"ty"`
	Code               string   `json:"cd"`
	OpeningPrice       *float64 `json:"op"`
	HighPrice          *float64 `json:"hp"`
	LowPrice           *float64 `json:"lp"`
	TradePrice         float64  `json:"tp"`
	PrevClosingPrice   float64  `json:"pcp"`
	Change             string   `json:"c"`
	ChangePrice        float64  `json:"cp"`
	SignedChangePrice  *float64 `json:"scp"`
	ChangeRate         *float64 `json:"cr"`
	SignedChangeRate   *float64 `json:"scr"`
	TradeVolume        float64  `json:"tv"`
	AccTradeVolume     *float64 `json:"atv"`
	AccTradeVolume24h  *float64 `json:"atv24h"`
	AccTradePrice      *float64 `json:"atp"`
	AccTradePrice24h   *float64 `json:"atp24h"`
	TradeDate          string   `json:"td"`
	TradeTime          string   `json:"ttm"`
	TradeTimestamp     int64    `json:"ttms"`
	AskBid             string   `json:"ab"`
	AccAskVolume       *float64 `json:"aav"`
	AccBidVolume       *float64 `json:"abv"`
	Highest52WeekPrice *float64 `json:"h52wp"`
	Highest52WeekDate  *string  `json:"h52wdt"`
	Lowest52WeekPrice  *float64 `json:"l52wp"`
	Lowest52WeekDate   *string  `json:"l52wdt"`
	TradeStatus        *string  `json:"ts"`
	MarketState        *string  `json:"ms"`
	MarketStateForIos  *string  `json:"msfi"`
	IsTradingSuspended *bool    `json:"its"`
	DelistingDate      *string  `json:"dd"`
	MarketWarning      *string  `json:"mw"`
	Timestamp          int64    `json:"tms"`
	StreamType         string   `json:"st"`
	SequentialId       *int64   `json:"sid"`
}
