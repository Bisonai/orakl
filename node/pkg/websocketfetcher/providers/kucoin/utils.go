package kucoin

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func RawDataToFeedData(raw SymbolSnapshotRaw, feedMap map[string]int32) *common.FeedData {
	snapshot := raw.Data.Data
	symbol := snapshot.Symbol
	id, _ := feedMap[symbol]

	timestamp := time.Unix(snapshot.Time/1000, 0)
	value := common.FormatFloat64Price(snapshot.Price)
	volume := snapshot.Volume

	return &common.FeedData{
		FeedID:    id,
		Value:     value,
		Timestamp: &timestamp,
		Volume:    volume,
	}
}
