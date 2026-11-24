package xt

const URL = "wss://stream.xt.com/public"

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     string   `json:"id"`
}

type Response struct {
	Topic string `json:"topic"`
	Event string `json:"event"`
	Data  struct {
		Symbol string `json:"s"`
		Time   int64  `json:"t"`
		Price  string `json:"c"`
		Volume string `json:"q"`
	} `json:"data"`
}

// DepthResponse is a depth@<symbol>,<levels> push. Its "t" is not the
// current time (hours behind when probed), so the receive time is used.
type DepthResponse struct {
	Topic string `json:"topic"`
	Data  struct {
		Symbol string     `json:"s"`
		Asks   [][]string `json:"a"`
		Bids   [][]string `json:"b"`
	} `json:"data"`
}
