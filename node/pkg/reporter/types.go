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
	SubmissionMsg           raft.MessageType = "submission"
	TOPIC_STRING                             = "orakl-offchain-aggregation-reporter"
	MESSAGE_BUFFER                           = 100
	DEVIATION_TIMEOUT                        = 5 * time.Second
	INITIAL_FAILURE_TIMEOUT                  = 50 * time.Millisecond
	MAX_RETRY                                = 3
	MAX_RETRY_DELAY                          = 500 * time.Millisecond
	SUBMIT_WITHOUT_PROOFS                    = "submit(address[] memory _feeds, int256[] memory _submissions)"
	SUBMIT_WITH_PROOFS                       = "submit(address[] memory _feeds, int256[] memory _answers, bytes[] memory _proofs, uint256[] memory _timestamps)"

	GET_SUBMISSIONS_QUERY        = `SELECT * FROM submission_addresses;`
	DEVIATION_THRESHOLD          = 0.05
	DEVIATION_ABSOLUTE_THRESHOLD = 0.1
	DECIMALS                     = 8
)

type SubmissionAddress struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	Address  string `db:"address"`
	Interval *int   `db:"interval"`
}

type SubmissionPair struct {
	LastSubmission int64 `db:"last_submission"`
	Address        common.Address
}

type App struct {
	Reporters []*Reporter
	Bus       *bus.MessageBus
	Host      host.Host
	Pubsub    *pubsub.PubSub
}

type Reporter struct {
	Raft               *raft.Raft
	KlaytnHelper       *helper.ChainHelper
	SubmissionPairs    map[string]SubmissionPair
	SubmissionInterval time.Duration

	contractAddress string

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type GlobalAggregate struct {
	Name      string    `db:"name" json:"name"`
	Value     int64     `db:"value" json:"value"`
	Round     int64     `db:"round" json:"round"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type Proof struct {
	Name  string `json:"name"`
	Round int64  `json:"round"`
	Proof []byte `json:"proofs"`
}

type PgsqlProof struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Round     int64     `db:"round" json:"round"`
	Proof     []byte    `db:"proof" json:"proof"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type SubmissionMessage struct {
	Submissions []GlobalAggregate `json:"submissions"`
}

func makeGetLatestGlobalAggregatesQuery(names []string) string {
	queryNames := make([]string, len(names))
	for i, name := range names {
		queryNames[i] = fmt.Sprintf("'%s'", name)
	}

	q := fmt.Sprintf(`
	SELECT ga.name, ga.value, ga.round, ga.timestamp
	FROM global_aggregates ga
	JOIN (
		SELECT name, MAX(round) as max_round
		FROM global_aggregates
		WHERE name IN (%s)
		GROUP BY name
	) subq ON ga.name = subq.name AND ga.round = subq.max_round;`, strings.Join(queryNames, ","))

	return q
}

func makeGetProofsQuery(aggregates []GlobalAggregate) string {
	placeHolders := make([]string, len(aggregates))
	for i, agg := range aggregates {
		placeHolders[i] = fmt.Sprintf("('%s', %d)", agg.Name, agg.Round)
	}

	return fmt.Sprintf("SELECT * FROM proofs WHERE (name, round) IN (%s);", strings.Join(placeHolders, ","))
}
