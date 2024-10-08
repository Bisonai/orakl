package mexc

import (
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedDataList(response BatchResponse, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	timestamp := time.UnixMilli(int64(response.Time))

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

		// mexc is using quote volume and volume in a opposite way
		volume, err := common.VolumeStringToFloat64(item.QuoteVolume)
		if err != nil {
			return feedDataList, err
		}

		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		feedDataList = append(feedDataList, feedData)
	}
	return feedDataList, nil
}
