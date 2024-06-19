package bybit

import (
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.UnixMilli(*data.Data.Time)
	value, err := common.PriceStringToFloat64(*data.Data.Price)
	if err != nil {
		return feedData, err
	}

	volume, err := common.VolumeStringToFloat64(*data.Data.Volume)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[*data.Data.Symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}

	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
