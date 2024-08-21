package gateio

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.Time, 0)
	price, err := common.PriceStringToFloat64(data.Result.Last)
	if err != nil {
		return feedData, err
	}

	volume, err := common.VolumeStringToFloat64(data.Result.BaseVolume)
	if err != nil {
		return feedData, err
	}

	key := strings.Replace(data.Result.CurrencyPair, "_", "-", 1)
	id, exists := feedMap[key]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}

	feedData.FeedID = id
	feedData.Value = price
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
