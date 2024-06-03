package kucoin

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func RawDataToFeedData(raw Raw, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(raw.Data.Time/1000, 0)
	value, err := common.PriceStringToFloat64(raw.Data.Price)
	if err != nil {
		return feedData, err
	}
	rawPair := strings.TrimPrefix(raw.Topic, "/market/ticker:")
	splitted := strings.Split(rawPair, "-")
	if len(splitted) < 2 {
		return feedData, fmt.Errorf("invalid feed name")
	}
	target := splitted[0]
	quote := splitted[1]

	id, exists := feedMap[strings.ToUpper(target)+"-"+strings.ToUpper(quote)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
