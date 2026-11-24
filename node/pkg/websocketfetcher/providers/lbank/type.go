package lbank

import "time"

const URL = "wss://www.lbkex.net/ws/V2/"

// SubscribeDelay paces the per-symbol subscriptions; lbank accepted 230 at
// 100 ms (and bursts of 30) in live tests.
const SubscribeDelay = 100 * time.Millisecond

type Subscription struct {
	Action    string `json:"action"`
	Subscribe string `json:"subscribe"`
	Pair      string `json:"pair"`
	// Depth is the number of levels of a "depth" subscription.
	Depth string `json:"depth,omitempty"`
}

// DepthResponse is a "depth" push (top levels); TS as in ticks.
type DepthResponse struct {
	Type  string `json:"type"`
	Pair  string `json:"pair"`
	TS    string `json:"TS"`
	Depth struct {
		Asks [][]string `json:"asks"`
		Bids [][]string `json:"bids"`
	} `json:"depth"`
}

type Response struct {
	Tick struct {
		ToCny    float64 `json:"to_cny"`
		High     float64 `json:"high"`
		Vol      float64 `json:"vol"`
		Low      float64 `json:"low"`
		Change   float64 `json:"change"`
		Usd      float64 `json:"usd"`
		ToUsd    float64 `json:"to_usd"`
		Dir      string  `json:"dir"`
		Turnover float64 `json:"turnover"`
		Latest   float64 `json:"latest"`
		Cny      float64 `json:"cny"`
	} `json:"tick"`
	Type   string `json:"type"`
	Pair   string `json:"pair"`
	Server string `json:"SERVER"`
	TS     string `json:"TS"`
}

type Ping struct {
	Action string `json:"action"`
	Ping   string `json:"ping"`
}

type Pong struct {
	Action string `json:"action"`
	Pong   string `json:"pong"`
}
