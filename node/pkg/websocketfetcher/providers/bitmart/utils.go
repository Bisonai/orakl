package bitmart

import (
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedData(response Response, feedMap map[string][]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, data := range response.Data {
		symbol := strings.ReplaceAll(data.Symbol, "_", "-")
		ids, exists := feedMap[symbol]
		if !exists {
			log.Warn().Str("Player", "Bitmart").Str("key", symbol).Msg("feed not found")
			continue
		}

		value, err := common.PriceStringToFloat64(data.Price)
		if err != nil {
			log.Warn().Str("Player", "Bitmart").Err(err).Msg("error in PriceStringToFloat64")
			continue
		}
		volume, err := common.VolumeStringToFloat64(data.Volume)
		if err != nil {
			log.Warn().Str("Player", "Bitmart").Err(err).Msg("error in VolumeStringToFloat64")
			continue
		}
		timestamp := time.UnixMilli(data.Time)

		for _, id := range ids {
			feedData := new(common.FeedData)
			feedData.FeedID = id
			feedData.Value = value
			feedData.Timestamp = &timestamp
			feedData.Volume = volume

			feedDataList = append(feedDataList, feedData)
		}
	}
	return feedDataList
}
