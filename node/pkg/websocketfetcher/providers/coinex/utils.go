package coinex

import (
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	for _, item := range data.Params {
		for key, value := range item {
			feedData := new(common.FeedData)
			id, exists := feedMap[key]
			if !exists {
				log.Warn().Str("Player", "Coinex").Str("key", key).Msg("feed not found")
				continue
			}
			price, err := common.PriceStringToFloat64(value.Last)
			if err != nil {
				log.Error().Str("Player", "Coinex").Err(err).Msg("error in PriceStringToFloat64")
				continue
			}
			volume, err := common.VolumeStringToFloat64(value.Volume)
			if err != nil {
				log.Error().Str("Player", "Coinex").Err(err).Msg("error in VolumeStringToFloat64")
				continue
			}
			timestamp := time.Now()
			feedData.FeedID = id
			feedData.Value = price
			feedData.Timestamp = &timestamp
			feedData.Volume = volume
			feedDataList = append(feedDataList, feedData)
		}
	}

	return feedDataList, nil
}
