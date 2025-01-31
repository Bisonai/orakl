package xt

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedData(response Response, feedMap map[string][]int32) ([]*common.FeedData, error) {
	symbol := strings.ToUpper(strings.ReplaceAll(response.Data.Symbol, "_", "-"))
	ids, exists := feedMap[symbol]
	if !exists {
		log.Warn().Str("Player", "xt").Str("raw", response.Data.Symbol).Str("key", symbol).Msg("feed not found")
		return nil, fmt.Errorf("feed not found from xt for symbol: %s", symbol)
	}
	timestamp := time.UnixMilli(response.Data.Time)
	value, err := common.PriceStringToFloat64(response.Data.Price)
	if err != nil {
		return nil, err
	}
	volume, err := common.VolumeStringToFloat64(response.Data.Volume)
	if err != nil {
		return nil, err
	}

	result := []*common.FeedData{}

	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume

		result = append(result, feedData)
	}

	return result, nil
}
