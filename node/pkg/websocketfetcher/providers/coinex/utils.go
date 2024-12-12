package coinex

import (
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string][]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	for _, item := range data.Data.StateList {

		id, exists := feedMap[item.Market]
		if !exists {
			log.Warn().Str("Player", "Coinex").Str("key", item.Market).Msg("feed not found")
			continue
		}
		price, err := common.PriceStringToFloat64(item.Last)
		if err != nil {
			log.Error().Str("Player", "Coinex").Err(err).Msg("error in PriceStringToFloat64")
			continue
		}
		volume, err := common.VolumeStringToFloat64(item.Volume)
		if err != nil {
			log.Error().Str("Player", "Coinex").Err(err).Msg("error in VolumeStringToFloat64")
			continue
		}
		timestamp := time.Now()

		for _, id := range id {
			feedData := new(common.FeedData)
			feedData.FeedID = id
			feedData.Value = price
			feedData.Timestamp = &timestamp
			feedData.Volume = volume
			feedDataList = append(feedDataList, feedData)
		}

	}

	return feedDataList, nil
}
