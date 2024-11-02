package kucoin

import (
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func RawDataToFeedData(raw SymbolSnapshotRaw, feedMap map[string][]int32) ([]*common.FeedData, error) {
	snapshot := raw.Data.Data
	symbol := snapshot.Symbol
	ids, ok := feedMap[symbol]
	if !ok {
		log.Warn().Str("Player", "Kucoin").Str("symbol", symbol).Msg("feed not found")
		return nil, fmt.Errorf("feed not found from kucoin for symbol: %s", symbol)
	}

	timestamp := time.UnixMilli(snapshot.Time)
	value := common.FormatFloat64Price(snapshot.Price)
	volume := snapshot.Volume

	result := []*common.FeedData{}
	for _, id := range ids {
		result = append(result, &common.FeedData{
			FeedID:    id,
			Value:     value,
			Timestamp: &timestamp,
			Volume:    volume,
		})
	}

	return result, nil
}
