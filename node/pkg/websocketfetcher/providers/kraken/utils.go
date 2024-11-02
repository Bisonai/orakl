package kraken

import (
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(response Response, feedMap map[string][]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, data := range response.Data {
		symbol := strings.ReplaceAll(data.Symbol, "/", "-")
		ids, exists := feedMap[symbol]
		if !exists {
			continue
		}

		value := common.FormatFloat64Price(data.Price)
		timestamp := time.Now()
		volume := data.Volume

		for _, id := range ids {
			feedData := new(common.FeedData)
			feedData.FeedID = id
			feedData.Value = value
			feedData.Timestamp = &timestamp
			feedData.Volume = volume
			feedDataList = append(feedDataList, feedData)
		}
	}
	return feedDataList
}
