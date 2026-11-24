package gateio

const URL = "wss://api.gateio.ws/ws/v4/"

type Subscription struct {
	Time    int64    `json:"time"`
	Channel string   `json:"channel"`
	Event   string   `json:"event"`
	Payload []string `json:"payload"`
}

type Response struct {
	Time    int64  `json:"time"`
	TimeMs  int64  `json:"time_ms"`
	Channel string `json:"channel"`
	Event   string `json:"event"`
	Result  struct {
		CurrencyPair     string `json:"currency_pair"`
		Last             string `json:"last"`
		LowestAsk        string `json:"lowest_ask"`
		HighestBid       string `json:"highest_bid"`
		ChangePercentage string `json:"change_percentage"`
		BaseVolume       string `json:"base_volume"`
		QuoteVolume      string `json:"quote_volume"`
		High24h          string `json:"high_24h"`
		Low24h           string `json:"low_24h"`
	} `json:"result"`
}

// BookTickerChannel pushes the best bid/ask on every change; t is in ms.
const BookTickerChannel = "spot.book_ticker"

type BookTickerResponse struct {
	Channel string `json:"channel"`
	Event   string `json:"event"`
	Result  struct {
		Timestamp    int64  `json:"t"`
		CurrencyPair string `json:"s"`
		Bid          string `json:"b"`
		Ask          string `json:"a"`
	} `json:"result"`
}
