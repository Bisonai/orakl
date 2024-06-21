package bingx

import (
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(response Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	symbol := response.Data.Symbol
	id, exists := feedMap[symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found for %s", symbol)
	}
	timestamp := time.UnixMilli(response.Data.EventTime)
	value := common.FormatFloat64Price(response.Data.Price)
	volume := response.Data.Volume

	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
