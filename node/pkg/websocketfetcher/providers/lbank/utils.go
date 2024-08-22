package lbank

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

const layout = "2006-01-02T15:04:05.000"

func ResponseToFeedData(data Response, feedMap map[string]int32) (*common.FeedData, error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	feedData := new(common.FeedData)

	timestampRaw, err := time.ParseInLocation(layout, data.TS, loc)
	if err != nil {
		return feedData, err
	}
	timestamp := timestampRaw.UTC()
	value := common.FormatFloat64Price(data.Tick.Latest)
	symbol := strings.ToUpper(strings.ReplaceAll(data.Pair, "_", "-"))
	volume := data.Tick.Vol

	id, exists := feedMap[symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp
	feedData.Volume = volume

	return feedData, nil
}
