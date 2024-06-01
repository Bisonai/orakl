package upbit

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(data.TradeTimestamp/1000, 0)
	price := common.FormatFloat64Price(data.TradePrice)
	splitted := strings.Split(data.Code, "-")
	base := splitted[1]
	quote := splitted[0]

	id, exists := feedMap[strings.ToUpper(base)+"-"+strings.ToUpper(quote)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedId = id
	feedData.Value = price
	feedData.Timestamp = &timestamp
	return feedData, nil
}
