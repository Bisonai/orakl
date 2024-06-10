package mexc

import (
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedDataList(response BatchResponse, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	timestamp := time.Unix(response.Time/1000, 0)

	for _, item := range response.Data {
		id, exists := feedMap[item.Symbol]
		if !exists {
			continue
		}

		feedData := new(common.FeedData)

		value, err := common.PriceStringToFloat64(item.Price)
		if err != nil {
			return feedDataList, err
		}
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedDataList = append(feedDataList, feedData)
	}
	return feedDataList, nil
}
