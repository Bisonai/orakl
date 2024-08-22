package crypto

import (
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedData := []*common.FeedData{}

	for _, tick := range data.Result.Data {
		if tick.LastTradePrice == nil {
			continue
		}
		timestamp := time.UnixMilli(tick.Timestamp)
		value, err := common.PriceStringToFloat64(*tick.LastTradePrice)
		if err != nil {
			log.Warn().Str("Player", "cryptodotcom").Str("priceValue", *tick.LastTradePrice).Err(err).Msg("failed to convert price string to float64")
			continue
		}

		volume, err := common.VolumeStringToFloat64(tick.Total24hTradeVolume)
		if err != nil {
			log.Warn().Str("Player", "cryptodotcom").Str("volumeValue", tick.Total24hTradeVolume).Err(err).Msg("failed to convert volume string to float64")
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
			FeedID:    id,
			Value:     value,
			Timestamp: &timestamp,
			Volume:    volume,
		})
	}

	return feedData, nil
}
