package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/raft"
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
	FUNCTION_STRING         = "submit(address[] memory _feeds, int256[] memory _submissions)"

	GET_LATEST_GLOBAL_AGGREGATES_QUERY = `
		SELECT ga.name, ga.value, ga.round, ga.timestamp, sa.address
		FROM global_aggregates ga
		JOIN (
			SELECT name, MAX(round) as max_round
			FROM global_aggregates
			GROUP BY name
		) subq ON ga.name = subq.name AND ga.round = subq.max_round
		INNER JOIN submission_addresses sa ON ga.name = sa.name;`
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
	KlaytnHelper    *helper.KlaytnHelper
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

func makeGetLatestGlobalAggregatesQuery(names []string) string {
	queryNames := make([]string, len(names))
	for i, name := range names {
		queryNames[i] = fmt.Sprintf("'%s'", name)
	}

	q := fmt.Sprintf(`
	SELECT ga.name, ga.value, ga.round
	FROM global_aggregates ga
	JOIN (
		SELECT name, MAX(round) as max_round
		FROM global_aggregates
		WHERE name IN (%s)
		GROUP BY name
	) subq ON ga.name = subq.name AND ga.round = subq.max_round;`, strings.Join(queryNames, ","))

	return q
}
