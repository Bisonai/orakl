package dal

import (
	"regexp"
	"sync"
	"time"
)

const (
	DefaultDalCheckInterval = 10 * time.Second
	DelayOffset             = 5 * time.Second
	AlarmOffset             = 3
	WsDelayThreshold        = 9 * time.Second
	WsPushThreshold         = 5 * time.Second
)

var (
	wsChan      = make(chan WsResponse, 30000)
	wsMsgChan   = make(chan string, 10000)
	updateTimes = &UpdateTimes{
		lastUpdates: make(map[string]time.Time),
	}
	re                = regexp.MustCompile(`\(([^)]+)\)`)
	trafficCheckQuery = `select count(1) from rest_calls where
timestamp > current_timestamp - INTERVAL '10 seconds' AND
api_key NOT IN (SELECT key from keys WHERE description IN ('test', 'sentinel', 'orakl_reporter'))`
	trafficThreshold = 100
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
