package coinone

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/wfetcher/common"
)

func DataToFeedData(data Data, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.Timestamp/1000, 0)
	value, err := common.PriceStringToFloat64(data.Last)
	if err != nil {
		return feedData, err
	}

	id, exists := feedMap[strings.ToUpper(data.TargetCurrency)+"-"+strings.ToUpper(data.QuoteCurrency)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedId = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
