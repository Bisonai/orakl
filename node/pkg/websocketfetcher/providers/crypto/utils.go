package crypto

import (
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedData := []*common.FeedData{}

	for _, tick := range data.Result.Data {
		if tick.LastTradePrice == nil {
			continue
		}
		timestamp := time.Unix(tick.Timestamp/1000, 0)
		value, err := common.PriceStringToFloat64(*tick.LastTradePrice)
		if err != nil {
			log.Warn().Str("Player", "cryptodotcom").Str("priceValue", *tick.LastTradePrice).Err(err).Msg("failed to convert price string to float64")
			continue
		}
		rawSymbol := strings.Split(tick.InstrumentName, "_")
		if len(rawSymbol) != 2 {
			log.Warn().Str("Player", "cryptodotcom").Str("rawSymbol", tick.InstrumentName).Msg("invalid instrument name")
			continue
		}

		base := rawSymbol[0]
		quote := rawSymbol[1]
		id, exists := feedMap[base+"-"+quote]
		if !exists {
			log.Warn().Str("Player", "cryptodotcom").Str("symbol", base+"-"+quote).Msg("feed not found")
			continue
		}

		feedData = append(feedData, &common.FeedData{
			FeedId:    id,
			Value:     value,
			Timestamp: &timestamp,
		})
	}

	return feedData, nil
}
