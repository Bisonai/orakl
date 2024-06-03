package coinbase

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TickerToFeedData(ticker Ticker, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp, err := time.Parse(time.RFC3339Nano, ticker.Time)
	if err != nil {
		log.Warn().Err(err).Msg("error in parsing time")
		timestamp = time.Now()
	}

	id, exists := feedMap[strings.ToUpper(ticker.ProductID)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}

	value, err := common.PriceStringToFloat64(ticker.Price)
	if err != nil {
		return feedData, err
	}

	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp

	return feedData, nil

}
