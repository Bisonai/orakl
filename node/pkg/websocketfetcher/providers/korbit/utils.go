package korbit

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func DataToFeedData(data Ticker, feedMap map[string][]int32) ([]*common.FeedData, error) {
	timestamp := time.UnixMilli(data.Timestamp)
	value, err := common.PriceStringToFloat64(data.Last)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(data.Volume)
	if err != nil {
		return nil, err
	}

	rawPair := strings.Split(data.CurrencyPair, "_")
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

		result = append(result, feedData)
	}

	return result, nil
}
