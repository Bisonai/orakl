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
	IgnoreKeys              = "test,sentinel,orakl_reporter"

	TrafficCheckQuery = `select count(1) from rest_calls where
timestamp > current_timestamp - INTERVAL '%d seconds' AND
api_key NOT IN (SELECT key from keys WHERE description IN (%s))`
	TrafficOldOffset    = 600
	TrafficRecentOffset = 10

	RestTimeout = 10 * time.Second
)

var (
	wsChan      = make(chan WsResponse, 30000)
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

type Count struct {
	Count int `db:"count"`
}
