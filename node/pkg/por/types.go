package por

import (
	"encoding/json"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/fetcher"
)

const (
	initialFailureTimeout = 500 * time.Millisecond
	maxRetry              = 3
	maxRetryDelay         = 5000 * time.Millisecond

	submitInterface           = "submit(uint256 _roundId, int256 _submission)"
	oracleRoundStateInterface = `function oracleRoundState(address _oracle, uint32 _queriedRoundId) external view returns (
            bool _eligibleToSubmit,
            uint32 _roundId,
            int256 _latestSubmission,
            uint64 _startedAt,
            uint64 _timeout,
            uint8 _oracleCount
    )`

	latestRoundDataInterface = `function latestRoundData() public view returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
	`
)

type app struct {
	entries    map[string]entry
	kaiaHelper *helper.ChainHelper
}

type entry struct {
	definition *fetcher.Definition
	adapter    adaptor
	aggregator aggregator
}

type feed struct {
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
	AdapterId  *int64          `json:"adapterId"`
}

type adaptor struct {
	Name     string `json:"name"`
	Feeds    []feed `json:"feeds"`
	Interval *int   `json:"interval"`
	Decimals int    `json:"decimals"`
}

type aggregator struct {
	Name              string  `json:"name"`
	Heartbeat         *int    `json:"heartbeat"`
	Address           string  `json:"address"`
	Threshold         float64 `json:"threshold"`
	AbsoluteThreshold float64 `json:"absoluteThreshold"`
}

type lastInfo struct {
	UpdatedAt *big.Int
	Answer    *big.Int
}

type urlEntry struct {
	adapterEndpoint, aggregatorEndpoint string
}
