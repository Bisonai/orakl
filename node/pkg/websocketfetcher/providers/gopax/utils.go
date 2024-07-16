package gopax

import (
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func InitialResponseToFeedData(initialData InitialResponse, feedMap map[string]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, data := range initialData.Data {
		feedData, err := TickerToFeedData(data, feedMap)
		if err != nil {
			continue
		}

		feedDataList = append(feedDataList, feedData)
	}

	return feedDataList
}

func TickerToFeedData(ticker Ticker, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	id, exists := feedMap[ticker.Name]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	timestamp := time.UnixMilli(ticker.Timestamp)
	feedData.Timestamp = &timestamp

	value := common.FormatFloat64Price(ticker.Price)
	feedData.Value = value
	feedData.Volume = ticker.Volume

	return feedData, nil
}
