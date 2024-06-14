package bitget

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}
	for _, tick := range data.Data {
		id, exists := feedMap[tick.InstId]
		if !exists {
			log.Error().Str("instId", tick.InstId).Msg("feed not found")
			continue
		}

		timestamp := time.Unix(0, tick.Ts*int64(time.Millisecond))
		value, err := common.PriceStringToFloat64(tick.Last)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert price string to float64")
			continue
		}

		volume, err := common.VolumeStringToFloat64(tick.BaseVolume)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert volume string to float64")
			continue
		}

		feedData := &common.FeedData{
			FeedID:    id,
			Value:     value,
			Timestamp: &timestamp,
			Volume:    volume,
		}
		feedDataList = append(feedDataList, feedData)
	}
	return feedDataList, nil
}
