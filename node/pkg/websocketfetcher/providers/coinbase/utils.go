package coinbase

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TickerToFeedData(ticker Ticker, feedMap map[string][]int32) ([]*common.FeedData, error) {
	ids, exists := feedMap[strings.ToUpper(ticker.ProductID)]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	timestamp, err := time.Parse(time.RFC3339Nano, ticker.Time)
	if err != nil {
		log.Warn().Err(err).Msg("error in parsing time")
		timestamp = time.Now()
	}

	value, err := common.PriceStringToFloat64(ticker.Price)
	if err != nil {
		return nil, err
	}

	volume, err := common.VolumeStringToFloat64(ticker.Volume24h)
	if err != nil {
		return nil, err
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
