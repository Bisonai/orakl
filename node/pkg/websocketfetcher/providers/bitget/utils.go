package bitget

import (
	"strconv"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, tick := range data.Data {
		id, exists := feedMap[tick.InstId]
		if !exists {
			log.Error().Str("instId", tick.InstId).Msg("feed not found")
			continue
		}
		value, err := common.PriceStringToFloat64(tick.Price)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert price string to float64")
			continue
		}
		volume, err := common.VolumeStringToFloat64(tick.Volume)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert volume string to float64")
			continue
		}
		timestampRaw, err := strconv.ParseInt(tick.Timestamp, 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert timestamp string to int64")
			continue
		}
		timestamp := time.UnixMilli(timestampRaw)
		feedData := &common.FeedData{
			FeedID:    id,
			Value:     value,
			Timestamp: &timestamp,
			Volume:    volume,
		}
		feedDataList = append(feedDataList, feedData)
	}
	return feedDataList
}
