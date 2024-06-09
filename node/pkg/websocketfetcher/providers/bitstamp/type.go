package bitstamp

const URL = "wss://ws.bitstamp.net"

type Subscription struct {
	Event string `json:"event"`
	Data  struct {
		Channel string `json:"channel"`
	} `json:"data"`
}

type Trade struct {
	ID             int64   `json:"id"`
	Amount         float64 `json:"amount"`
	AmountStr      string  `json:"amount_str"`
	Price          float64 `json:"price"`
	PriceStr       string  `json:"price_str"`
	Type           int     `json:"type"`
	Timestamp      string  `json:"timestamp"`
	Microtimestamp string  `json:"microtimestamp"`
	BuyOrderID     int64   `json:"buy_order_id"`
	SellOrderID    int64   `json:"sell_order_id"`
}

type TradeEvent struct {
	Channel string `json:"channel"`
	Data    Trade  `json:"data"`
	Event   string `json:"event"`
}
