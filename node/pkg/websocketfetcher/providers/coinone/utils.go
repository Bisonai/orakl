package coinone

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func DataToFeedData(data Data, feedMap map[string][]int32) ([]*common.FeedData, error) {
	ids, exists := feedMap[strings.ToUpper(data.TargetCurrency)+"-"+strings.ToUpper(data.QuoteCurrency)]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	timestamp := time.UnixMilli(data.Timestamp)
	value, err := common.PriceStringToFloat64(data.Last)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(data.TargetVolume)
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
