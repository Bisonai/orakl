package bybit

import (
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.Ts/1000, 0)
	value, err := common.PriceStringToFloat64(*data.Data.LastPrice)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[*data.Data.Symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedId = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
