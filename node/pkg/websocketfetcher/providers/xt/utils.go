package xt

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(response Response, feedMap map[string][]int32) ([]*common.FeedData, error) {
	symbol := strings.ToUpper(strings.ReplaceAll(response.Data.Symbol, "_", "-"))
	ids, exists := feedMap[symbol]
	if !exists {
		return nil, fmt.Errorf("feed not found")
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
