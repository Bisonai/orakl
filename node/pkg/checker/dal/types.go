package dal

import (
	"sync"
	"time"
)

const (
	DefaultDalCheckInterval = 10 * time.Second
	DelayOffset             = 5 * time.Second
	AlarmOffsetPerPair      = 10
	AlarmOffsetInTotal      = 3
	WsDelayThreshold        = 9 * time.Second
	WsPushThreshold         = 5 * time.Second
	NetworkDelayThreshold   = 6 * time.Second

	RestTimeout = 10 * time.Second
)

var (
	updateTimes = &UpdateTimes{
		lastUpdates: make(map[string]time.Time),
	}
)

type WsResponse struct {
	Symbol        string `json:"symbol"`
	AggregateTime string `json:"aggregateTime"`
}

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Config struct {
	Name           string `json:"name"`
	SubmitInterval *int   `json:"submitInterval"`
}

type OutgoingSubmissionData struct {
	Symbol        string `json:"symbol"`
	Value         string `json:"value"`
	AggregateTime string `json:"aggregateTime"`
	Proof         string `json:"proof"`
	FeedHash      string `json:"feedHash"`
	Decimals      string `json:"decimals"`
}

type UpdateTimes struct {
	lastUpdates map[string]time.Time
	mu          sync.RWMutex
}
