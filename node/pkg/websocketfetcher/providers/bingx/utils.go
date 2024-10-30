package bingx

import (
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(response Response, feedMap map[string][]int32) ([]*common.FeedData, error) {

	symbol := response.Data.Symbol
	ids, exists := feedMap[symbol]
	if !exists {
		return nil, fmt.Errorf("feed not found for %s", symbol)
	}
	timestamp := time.UnixMilli(response.Data.EventTime)
	value := common.FormatFloat64Price(response.Data.Price)
	volume := response.Data.Volume

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
