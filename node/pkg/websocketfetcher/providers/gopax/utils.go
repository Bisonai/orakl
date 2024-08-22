package gopax

import (
	"errors"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func InitialResponseToFeedData(initialData InitialResponse, feedMap map[string]int32) []*common.FeedData {
	feedDataList := []*common.FeedData{}
	for _, data := range initialData.Data {
		feedData, err := TickerToFeedData(data, feedMap)
		if err != nil {
			if !errors.Is(err, errorSentinel.ErrFetcherFeedNotFound) {
				log.Warn().Str("Player", "Gopax").Err(err).Msg("error in TickerToFeedData")
			}
			continue
		}

		feedDataList = append(feedDataList, feedData)
	}

	return feedDataList
}

func TickerToFeedData(ticker Ticker, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	id, exists := feedMap[ticker.Name]
	if !exists {
		return feedData, errorSentinel.ErrFetcherFeedNotFound
	}
	feedData.FeedID = id
	timestamp := time.UnixMilli(ticker.Timestamp)
	feedData.Timestamp = &timestamp

	value := common.FormatFloat64Price(ticker.Price)
	feedData.Value = value
	feedData.Volume = ticker.Volume

	return feedData, nil
}
