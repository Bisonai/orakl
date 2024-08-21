package bybit

import (
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	id, exists := feedMap[data.Data.Symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}

	timestamp := time.UnixMilli(*data.Timestamp)
	value, err := common.PriceStringToFloat64(data.Data.Price)
	if err != nil {
		return feedData, err
	}

	volume, err := common.VolumeStringToFloat64(data.Data.Volume)
	if err != nil {
		return feedData, err
	}

	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
