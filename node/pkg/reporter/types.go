package reporter

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/klaytn_helper"
	"github.com/klaytn/klaytn/common"
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

	GET_LATEST_GLOBAL_AGGREGATE_QUERY = `
		SELECT value, round
		FROM global_aggregates
		WHERE name = @name
		ORDER BY round DESC
		LIMIT 1;
	`
)

type SubmissionAddress struct {
	Id      int    `db:"id"`
	Name    string `db:"name"`
	Address string `db:"address"`
}

type SubmissionPair struct {
	LastSubmission int64 `db:"last_submission"`
	Address        common.Address
}

type App struct {
	Reporter *ReporterNode
	Bus      *bus.MessageBus
	Host     host.Host
	Pubsub   *pubsub.PubSub
}

type ReporterNode struct {
	Raft            *raft.Raft
	TxHelper        *klaytn_helper.TxHelper
	SubmissionPairs map[string]SubmissionPair

	contractAddress string

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type GlobalAggregate struct {
	Name  string `db:"name" json:"name"`
	Value int64  `db:"value" json:"value"`
	Round int64  `db:"round" json:"round"`
}
