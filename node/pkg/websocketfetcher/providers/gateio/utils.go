package gateio

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string][]int32) ([]*common.FeedData, error) {

	timestamp := time.Unix(data.Time, 0)
	price, err := common.PriceStringToFloat64(data.Result.Last)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(data.Result.BaseVolume)
	if err != nil {
		return nil, err
	}

	key := strings.Replace(data.Result.CurrencyPair, "_", "-", 1)
	ids, exists := feedMap[key]
	if !exists {
		return nil, fmt.Errorf("feed not found from gateio for symbol: %s", key)
	}

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = price
		feedData.Timestamp = &timestamp
		feedData.Volume = volume

		result = append(result, feedData)
	}

	return result, nil
}
