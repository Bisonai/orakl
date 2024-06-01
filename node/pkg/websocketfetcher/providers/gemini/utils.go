package gemini

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TradeResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	timestamp := time.Unix(*data.TimestampMs/1000, 0)
	for _, event := range data.Events {
		feedData := new(common.FeedData)
		id, exists := feedMap[event.Symbol]
		if !exists {
			log.Warn().Str("Player", "Gemini").Str("key", event.Symbol).Msg("feed not found")
			continue
		}

		price, err := common.PriceStringToFloat64(event.Price)
		if err != nil {
			log.Warn().Str("Player", "Gemini").Err(err).Msg("error in PriceStringToFloat64")
			continue
		}
		feedData.FeedId = id
		feedData.Value = price
		feedData.Timestamp = &timestamp
		feedDataList = append(feedDataList, feedData)
	}

	return feedDataList, nil
}
