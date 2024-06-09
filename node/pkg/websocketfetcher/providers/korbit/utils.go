package korbit

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func DataToFeedData(data Ticker, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.Timestamp/1000, 0)
	value, err := common.PriceStringToFloat64(data.Last)
	if err != nil {
		return feedData, err
	}
	rawPair := strings.Split(data.CurrencyPair, "_")
	if len(rawPair) < 2 {
		return feedData, fmt.Errorf("invalid feed name")
	}
	target := rawPair[0]
	quote := rawPair[1]

	id, exists := feedMap[strings.ToUpper(target)+"-"+strings.ToUpper(quote)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
