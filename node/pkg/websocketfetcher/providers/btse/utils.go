package btse

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedData := []*common.FeedData{}

	for _, ticker := range data.Data {
		timestamp := time.Unix(ticker.Timestamp/1000, 0)
		price := common.FormatFloat64Price(ticker.Price)
		id, exists := feedMap[ticker.Symbol]
		if !exists {
			log.Warn().Str("Player", "btse").Str("symbol", ticker.Symbol).Msg("feed not found")
			continue
		}

		feedData = append(feedData, &common.FeedData{
			FeedID:    id,
			Value:     price,
			Timestamp: &timestamp,
		})
	}
	return feedData, nil
}
