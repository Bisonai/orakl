package kucoin

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func RawDataToFeedData(raw MarketSnapshotRaw, feedMap map[string]int32) []*common.FeedData {
	feedData := []*common.FeedData{}

	data := raw.Data.Data
	for _, snapshot := range data {
		symbol := snapshot.Symbol
		id, exists := feedMap[symbol]
		if !exists {
			continue
		}
		timestamp := time.Unix(snapshot.Time/1000, 0)
		value := common.FormatFloat64Price(snapshot.Price)
		volume := snapshot.Volume

		feedDataItem := common.FeedData{
			FeedID:    id,
			Value:     value,
			Timestamp: &timestamp,
			Volume:    volume,
		}

		feedData = append(feedData, &feedDataItem)
	}

	return feedData
}
