package por

import (
	"encoding/json"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/fetcher"
)

const (
	DECIMALS            = 4
	DEVIATION_THRESHOLD = 0.0001
	ABSOLUTE_THRESHOLD  = 0.1

	INITIAL_FAILURE_TIMEOUT = 500 * time.Millisecond
	MAX_RETRY               = 3
	MAX_RETRY_DELAY         = 5000 * time.Millisecond

	SUBMIT_FUNCTION_STRING = "submit(uint256 _roundId, int256 _submission)"
	READ_ROUND_ID          = `function oracleRoundState(address _oracle, uint32 _queriedRoundId) external view returns (
            bool _eligibleToSubmit,
            uint32 _roundId,
            int256 _latestSubmission,
            uint64 _startedAt,
            uint64 _timeout,
            uint8 _oracleCount
    )`

	READ_LATEST_ROUND_DATA = `function latestRoundData() public view returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
	`
)

type App struct {
	Name            string
	Definition      *fetcher.Definition
	FetchInterval   time.Duration
	SubmitInterval  time.Duration
	KaiaHelper      *helper.ChainHelper
	ContractAddress string
}

type FeedModel struct {
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
	AdapterId  *int64          `json:"adapterId"`
}

type AdapterModel struct {
	Name     string      `json:"name"`
	Feeds    []FeedModel `json:"feeds"`
	Interval *int        `json:"interval"`
}

type AggregatorModel struct {
	Name      string `json:"name"`
	Heartbeat *int   `json:"heartbeat"`
	Address   string `json:"address"`
}

type SubmissionModel struct {
	Name  string    `json:"name"`
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type LastInfo struct {
	UpdatedAt *big.Int
	Answer    *big.Int
}
