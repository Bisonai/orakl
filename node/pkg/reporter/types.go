package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common"
	"bisonai.com/orakl/node/pkg/raft"
	klaytnCommon "github.com/klaytn/klaytn/common"
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
	SUBMIT_WITH_PROOFS                       = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	GET_ONCHAIN_WHITELIST                    = "getAllOracles() public view returns (address[] memory)"

	GET_REPORTER_CONFIGS = `SELECT name, id, submit_interval, aggregate_interval FROM configs;`

	DEVIATION_THRESHOLD          = 0.05
	DEVIATION_ABSOLUTE_THRESHOLD = 0.1
	DECIMALS                     = 8
)

type Config struct {
	ID                int32  `db:"id"`
	Name              string `db:"name"`
	SubmitInterval    *int   `db:"submit_interval"`
	AggregateInterval *int   `db:"aggregate_interval"`
}

type SubmissionPair struct {
	LastSubmission int32  `db:"last_submission"`
	Name           string `db:"name"`
}

type App struct {
	Reporters []*Reporter
	Bus       *bus.MessageBus
	Host      host.Host
	Pubsub    *pubsub.PubSub
}

type JobType int

const (
	ReportJob JobType = iota
	DeviationJob
)

type ReporterConfig struct {
	Host            host.Host
	Ps              *pubsub.PubSub
	Configs         []Config
	Interval        int
	ContractAddress string
	CachedWhitelist []klaytnCommon.Address
	JobType         JobType
}

type ReporterOption func(*ReporterConfig)

func WithHost(h host.Host) ReporterOption {
	return func(c *ReporterConfig) {
		c.Host = h
	}
}

func WithPubsub(ps *pubsub.PubSub) ReporterOption {
	return func(c *ReporterConfig) {
		c.Ps = ps
	}
}

func WithConfigs(configs []Config) ReporterOption {
	return func(c *ReporterConfig) {
		c.Configs = configs
	}
}

func WithInterval(interval int) ReporterOption {
	return func(c *ReporterConfig) {
		c.Interval = interval
	}
}

func WithContractAddress(address string) ReporterOption {
	return func(c *ReporterConfig) {
		c.ContractAddress = address
	}
}

func WithCachedWhitelist(whitelist []klaytnCommon.Address) ReporterOption {
	return func(c *ReporterConfig) {
		c.CachedWhitelist = whitelist
	}
}

func WithJobType(jobType JobType) ReporterOption {
	return func(c *ReporterConfig) {
		c.JobType = jobType
	}
}

type Reporter struct {
	Raft               *raft.Raft
	KlaytnHelper       *helper.ChainHelper
	SubmissionPairs    map[int32]SubmissionPair
	SubmissionInterval time.Duration
	CachedWhitelist    []klaytnCommon.Address

	contractAddress string

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool
}

type GlobalAggregate common.GlobalAggregate

type Proof struct {
	ConfigID int32  `json:"configId"`
	Round    int32  `json:"round"`
	Proof    []byte `json:"proofs"`
}

type PgsqlProof struct {
	ID        int32     `db:"id" json:"id"`
	ConfigID  int32     `db:"config_id" json:"configId"`
	Round     int32     `db:"round" json:"round"`
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
