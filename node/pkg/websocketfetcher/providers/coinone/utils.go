package coinone

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func DataToFeedData(data Data, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.Timestamp/1000, 0)
	value, err := common.PriceStringToFloat64(data.Last)
	if err != nil {
		return feedData, err
	}

	volume, err := common.VolumeStringToFloat64(data.TargetVolume)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[strings.ToUpper(data.TargetCurrency)+"-"+strings.ToUpper(data.QuoteCurrency)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume
	return feedData, nil
}
