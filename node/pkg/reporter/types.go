package reporter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/klaytn/klaytn/common"
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

	MAX_REPORT_BATCH_SIZE = 50
	DEVIATION_INTERVAL    = 2000

	DEVIATION_ABSOLUTE_THRESHOLD = 0.1
	DECIMALS                     = 8

	MAX_DEVIATION_THRESHOLD = 0.01
	MIN_DEVIATION_THRESHOLD = 0.05
	MIN_INTERVAL            = 15
	MAX_INTERVAL            = 3600
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
	Reporters   []*Reporter
	chainHelper *helper.ChainHelper

	WsHelper   *wss.WebsocketHelper
	LatestData sync.Map
}

type JobType int

const (
	ReportJob JobType = iota
	DeviationJob
)

type ReporterConfig struct {
	Configs         []Config
	Interval        int
	ContractAddress string
	CachedWhitelist []common.Address
	JobType         JobType
	DalApiKey       string
	DalWsEndpoint   string
}

type ReporterOption func(*ReporterConfig)

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
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

func WithCachedWhitelist(whitelist []common.Address) ReporterOption {
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
	KaiaHelper         *helper.ChainHelper
	SubmissionPairs    map[int32]SubmissionPair
	SubmissionInterval time.Duration
	CachedWhitelist    []common.Address

	contractAddress    string
	deviationThreshold float64

	nodeCtx    context.Context
	nodeCancel context.CancelFunc
	isRunning  bool

	LatestData *sync.Map
}

type GlobalAggregate types.GlobalAggregate

type Proof types.Proof

type RawSubmissionData struct {
	Value         string   `json:"value"`
	AggregateTime string   `json:"aggregateTime"`
	Proof         []byte   `json:"proof"`
	FeedHash      [32]byte `json:"feedHash"`
}
type SubmissionData struct {
	Value         int64    `json:"value"`
	AggregateTime int64    `json:"aggregateTime"`
	Proof         []byte   `json:"proof"`
	FeedHash      [32]byte `json:"feedHash"`
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
