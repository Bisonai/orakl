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
	SUBMIT_WITH_PROOFS                       = "submit(address[] memory _feeds, int256[] memory _answers, uint256[] memory _timestamps, bytes[] memory _proofs)"
	GET_ONCHAIN_WHITELIST                    = "getAllOracles() public view returns (address[] memory)"

	GET_REPORTER_CONFIGS = `SELECT name, id, address, submit_interval FROM configs;`

	DEVIATION_THRESHOLD          = 0.05
	DEVIATION_ABSOLUTE_THRESHOLD = 0.1
	DECIMALS                     = 8
)

type ReporterConfig struct {
	ID             int32  `db:"id"`
	Name           string `db:"name"`
	Address        string `db:"address"`
	SubmitInterval *int   `db:"submit_interval"`
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
	SubmissionPairs    map[int32]SubmissionPair
	SubmissionInterval time.Duration
	CachedWhitelist    []common.Address

	contractAddress string

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type GlobalAggregate struct {
	ConfigID  int32     `db:"config_id" json:"configId"`
	Value     int64     `db:"value" json:"value"`
	Round     int64     `db:"round" json:"round"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type Proof struct {
	ConfigID int32  `json:"configId"`
	Round    int64  `json:"round"`
	Proof    []byte `json:"proofs"`
}

type PgsqlProof struct {
	ID        int32     `db:"id" json:"id"`
	ConfigID  int32     `db:"config_id" json:"configId"`
	Round     int64     `db:"round" json:"round"`
	Proof     []byte    `db:"proof" json:"proof"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

type SubmissionMessage struct {
	Submissions []GlobalAggregate `json:"submissions"`
}

func makeGetLatestGlobalAggregatesQuery(configIds []int32) string {
	queryConfigIds := make([]string, len(configIds))
	for i, id := range configIds {
		queryConfigIds[i] = fmt.Sprintf("'%d'", id)
	}

	q := fmt.Sprintf(`
	SELECT ga.config_id, ga.value, ga.round, ga.timestamp
	FROM global_aggregates ga
	JOIN (
		SELECT config_id, MAX(round) as max_round
		FROM global_aggregates
		WHERE config_id IN (%s)
		GROUP BY config_id
	) subq ON ga.config_id = subq.config_id AND ga.round = subq.max_round;`, strings.Join(queryConfigIds, ","))

	return q
}

func makeGetProofsQuery(aggregates []GlobalAggregate) string {
	placeHolders := make([]string, len(aggregates))
	for i, agg := range aggregates {
		placeHolders[i] = fmt.Sprintf("('%d', %d)", agg.ConfigID, agg.Round)
	}

	return fmt.Sprintf("SELECT * FROM proofs WHERE (config_id, round) IN (%s);", strings.Join(placeHolders, ","))
}
