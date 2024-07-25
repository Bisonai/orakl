package reporter

import (
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/klaytn/klaytn/common"
)

const (
	SUBMIT_WITH_PROOFS    = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	GET_ONCHAIN_WHITELIST = "getAllOracles() public view returns (address[] memory)"

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
	LastSubmission int64
	Name           string
}

type App struct {
	Reporters []*Reporter

	WsHelper   *wss.WebsocketHelper
	LatestData *sync.Map
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
	KaiaHelper      *helper.ChainHelper
	LatestData      *sync.Map
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

func WithKaiaHelper(chainHelper *helper.ChainHelper) ReporterOption {
	return func(c *ReporterConfig) {
		c.KaiaHelper = chainHelper
	}
}

func WithLatestData(latestData *sync.Map) ReporterOption {
	return func(c *ReporterConfig) {
		c.LatestData = latestData
	}
}

type Reporter struct {
	KaiaHelper         *helper.ChainHelper
	SubmissionPairs    map[int32]SubmissionPair
	SubmissionInterval time.Duration
	CachedWhitelist    []common.Address

	contractAddress    string
	deviationThreshold float64

	LatestData *sync.Map // map[symbol]SubmissionData
	Job        func() error
}

type GlobalAggregate types.GlobalAggregate

type RawSubmissionData struct {
	Value         string `json:"value"`
	AggregateTime string `json:"aggregateTime"`
	Proof         string `json:"proof"`
	FeedHash      string `json:"feedHash"`
}
type SubmissionData struct {
	Value         int64    `json:"value"`
	AggregateTime int64    `json:"aggregateTime"`
	Proof         []byte   `json:"proof"`
	FeedHash      [32]byte `json:"feedHash"`
}
