package okx

import (
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedData(response Response, feedMap map[string]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, data := range response.Data {
		id, exists := feedMap[data.InstId]
		if !exists {
			continue
		}

		feedData := new(common.FeedData)
		value, err := common.PriceStringToFloat64(data.Price)
		if err != nil {
			log.Error().Err(err).Str("Player", "OKX").Msg("error in PriceStringToFloat64")
			continue
		}
		intTimestamp, err := strconv.ParseInt(data.Timestamp, 10, 64)
		if err != nil {
			log.Error().Err(err).Str("Player", "OKX").Msg("error in strconv.ParseInt")
			continue
		}
		timestamp := time.UnixMilli(intTimestamp)
		volume, err := common.VolumeStringToFloat64(data.Volume)
		if err != nil {
			log.Error().Err(err).Str("Player", "OKX").Msg("error in VolumeStringToFloat64")
			continue
		}

		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		feedDataList = append(feedDataList, feedData)
	}
	return feedDataList
}
