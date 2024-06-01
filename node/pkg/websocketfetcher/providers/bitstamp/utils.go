package bitstamp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
)

func TradeEventToFeedData(data TradeEvent, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)
	rawTimestamp, err := strconv.ParseInt(data.Data.Microtimestamp, 10, 64)
	if err != nil {
		return feedData, err
	}
	timestamp := time.Unix(0, rawTimestamp*int64(time.Microsecond))
	value := common.FormatFloat64Price(data.Data.Price)
	rawSymbol := strings.Split(data.Channel, "_")[2]
	id, exists := feedMap[strings.ToUpper(rawSymbol)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedId = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	return feedData, nil
}
