package gopax

import (
	"errors"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func InitialResponseToFeedData(initialData InitialResponse, feedMap map[string][]int32) []*common.FeedData {
	result := []*common.FeedData{}
	for _, data := range initialData.Data {
		feedDataList, err := TickerToFeedData(data, feedMap)
		if err != nil {
			if !errors.Is(err, errorSentinel.ErrFetcherFeedNotFound) {
				log.Warn().Str("Player", "Gopax").Err(err).Msg("error in TickerToFeedData")
			}
			continue
		}

		result = append(result, feedDataList...)
	}

	return result
}

func TickerToFeedData(ticker Ticker, feedMap map[string][]int32) ([]*common.FeedData, error) {
	ids, exists := feedMap[ticker.Name]
	if !exists {
		return nil, errorSentinel.ErrFetcherFeedNotFound
	}
	timestamp := time.UnixMilli(ticker.Timestamp)
	value := ticker.Price

	result := []*common.FeedData{}
	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Timestamp = &timestamp
		feedData.Value = value
		feedData.Volume = ticker.Volume
		result = append(result, feedData)
	}

	return result, nil
}
