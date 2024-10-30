package binance

import (
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func TickerToFeedData(miniTicker MiniTicker, feedMap map[string][]int32) ([]*common.FeedData, error) {

	timestamp := time.UnixMilli(miniTicker.EventTime)
	value, err := common.PriceStringToFloat64(miniTicker.Price)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(miniTicker.Volume)
	if err != nil {
		return nil, err
	}

	ids, exists := feedMap[miniTicker.Symbol]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		result = append(result, feedData)
	}

	return result, nil
}
