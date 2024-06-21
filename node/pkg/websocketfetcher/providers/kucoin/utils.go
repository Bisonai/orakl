package kucoin

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func RawDataToFeedData(raw SymbolSnapshotRaw, feedMap map[string]int32) *common.FeedData {
	snapshot := raw.Data.Data
	symbol := snapshot.Symbol
	id := feedMap[symbol]

	timestamp := time.UnixMilli(snapshot.Time)
	value := common.FormatFloat64Price(snapshot.Price)
	volume := snapshot.Volume

	return &common.FeedData{
		FeedID:    id,
		Value:     value,
		Timestamp: &timestamp,
		Volume:    volume,
	}
}
