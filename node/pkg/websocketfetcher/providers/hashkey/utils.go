package hashkey

import (
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func ResponseToFeedDataList(response Response, feedMap map[string][]int32) ([]*common.FeedData, error) {
	d := response.Data

	ids, exists := feedMap[d.Symbol]
	if !exists {
		return nil, fmt.Errorf("feed not found from hashkey for symbol: %s", d.Symbol)
	}

	value, err := common.PriceStringToFloat64(d.Price)
	if err != nil {
		return nil, err
	}

	// d.Volume is `v`, the base-asset volume; `qv` is the quote (USDT) turnover.
	// FeedData.Volume is the base amount everywhere else, so use v (not qv).
	volume, err := common.VolumeStringToFloat64(d.Volume)
	if err != nil {
		return nil, err
	}

	timestamp := time.UnixMilli(d.Timestamp)

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		result = append(result, feedData)
	}

	return result, nil
}
