package korbit

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func RawToFeedData(raw Raw, feedMap map[string][]int32) ([]*common.FeedData, error) {
	timestamp := time.UnixMilli(raw.Timestamp)
	value, err := common.PriceStringToFloat64(raw.Data.Close)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(raw.Data.Volume)
	if err != nil {
		return nil, err
	}

	rawPair := strings.Split(raw.Symbol, "_")
	if len(rawPair) < 2 {
		return nil, fmt.Errorf("invalid feed name")
	}
	target := rawPair[0]
	quote := rawPair[1]

	ids, exists := feedMap[strings.ToUpper(target)+"-"+strings.ToUpper(quote)]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp
		feedData.Volume = volume
		// The ticker carries the best bid/ask and is re-sent regularly.
		if bid, err := common.PriceStringToFloat64(raw.Data.BestBidPrice); err == nil {
			if ask, err := common.PriceStringToFloat64(raw.Data.BestAskPrice); err == nil {
				common.SetTickerBook(feedData, bid, ask, timestamp)
			}
		}

		result = append(result, feedData)
	}

	return result, nil
}
