package reporter

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/klaytn_helper"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	TOPIC_STRING            = "orakl-offchain-aggregation-reporter"
	MESSAGE_BUFFER          = 100
	LEADER_TIMEOUT          = 5 * time.Second
	INITIAL_FAILURE_TIMEOUT = 50 * time.Millisecond
	MAX_RETRY               = 3
	MAX_RETRY_DELAY         = 500 * time.Millisecond
	FUNCTION_STRING         = "batchSubmit(address[] memory _addresses, int256[] memory _prices)"

	GET_LATEST_GLOBAL_AGGREGATES_QUERY = `
		SELECT ga.name, ga.value, ga.round, ga.timestamp, sa.address
		FROM global_aggregates ga
		JOIN (
			SELECT name, MAX(round) as max_round
			FROM global_aggregates
			GROUP BY name
		) subq ON ga.name = subq.name AND ga.round = subq.max_round
		INNER JOIN submission_addresses sa ON ga.name = sa.name;
	`
)

type App struct {
	Reporter *ReporterNode
	Bus      *bus.MessageBus
	Host     host.Host
	Pubsub   *pubsub.PubSub
}

type ReporterNode struct {
	Raft     *raft.Raft
	TxHelper *klaytn_helper.TxHelper

	lastSubmissions map[string]int64
	contractAddress string

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type GlobalAggregateBase struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Round     int64     `db:"round"`
	Timestamp time.Time `db:"timestamp"`
}

type GlobalAggregate struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Round     int64     `db:"round"`
	Timestamp time.Time `db:"timestamp"`
	Address   string    `db:"address"`
}
