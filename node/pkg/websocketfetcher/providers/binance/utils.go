package binance

import (
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func TickerToFeedData(miniTicker MiniTicker, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)
	timestamp := time.Unix(miniTicker.EventTime/1000, 0)
	value, err := common.PriceStringToFloat64(miniTicker.Price)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[miniTicker.Symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedId = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
