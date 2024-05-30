package common

import (
	"time"
)

type Proxy struct {
	ID       int64   `db:"id"`
	Protocol string  `db:"protocol"`
	Host     string  `db:"host"`
	Port     int     `db:"port"`
	Location *string `db:"location"`
}

type FeedData struct {
	FeedID    int32      `db:"feed_id"`
	Value     float64    `db:"value"`
	Timestamp *time.Time `db:"timestamp"`
}

type LocalAggregate struct {
	ConfigID  int32     `db:"config_id" json:"configId"`
	Value     int64     `db:"value" json:"value"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type GlobalAggregate struct {
	ConfigID  int32     `db:"config_id" json:"configId"`
	Value     int64     `db:"value" json:"value"`
	Round     int32     `db:"round" json:"round"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type Proof struct {
	ConfigID int32  `json:"configId"`
	Round    int32  `json:"round"`
	Proof    []byte `json:"proofs"`
}

type Config struct {
	ID                int32  `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	FetchInterval     int    `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval int    `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    int    `db:"submit_interval" json:"submitInterval"`
}
