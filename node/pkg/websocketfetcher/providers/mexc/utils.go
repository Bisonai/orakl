package mexc

import (
	"fmt"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(response Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestampRaw, err := strconv.ParseInt(response.Data.Time, 10, 64)
	if err != nil {
		return feedData, err
	}

	timestamp := time.Unix(timestampRaw/1000, 0)
	value, err := common.PriceStringToFloat64(response.Data.Price)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[response.Data.Symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
