package upbit

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string][]int32) ([]*common.FeedData, error) {

	timestamp := time.UnixMilli(data.TradeTimestamp)
	price := common.FormatFloat64Price(data.TradePrice)

	volume := data.AccTradeVolume24h

	splitted := strings.Split(data.Code, "-")
	base := splitted[1]
	quote := splitted[0]

	ids, exists := feedMap[strings.ToUpper(base)+"-"+strings.ToUpper(quote)]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = price
		feedData.Timestamp = &timestamp
		feedData.Volume = *volume

		result = append(result, feedData)
	}

	return result, nil
}
