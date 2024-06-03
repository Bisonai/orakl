package bithumb

import (
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

const layout = "2006-01-02 15:04:05.999999"

const dateLayout = "20060102"
const timeLayout = "150405"

func TransactionResponseToFeedDataList(data TransactionResponse, feedMap map[string]int32) ([]*common.FeedData, error) {
	feedData := []*common.FeedData{}
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Error().Err(err).Msg("error in bithumb load location")
		return feedData, err
	}

	for _, transaction := range data.Content.List {
		rawTime := transaction.ContDtm

		timestamp, err := time.ParseInLocation(layout, rawTime, loc)
		if err != nil {
			log.Error().Err(err).Msg("error in bithumb.TransactionResponseToFeedDataList")
			continue
		}
		timestamp = timestamp.UTC()

		price, err := common.PriceStringToFloat64(transaction.ContPrice)
		if err != nil {
			log.Error().Err(err).Msg("error in bithumb.TransactionResponseToFeedDataList")
			continue
		}

		splitted := strings.Split(transaction.Symbol, "_")
		symbol := splitted[0] + "-" + splitted[1]

		id, exists := feedMap[symbol]
		if !exists {
			log.Warn().Str("Player", "bithumb").Str("symbol", symbol).Msg("feed not found")
			continue
		}

		feedData = append(feedData, &common.FeedData{
			FeedID:    id,
			Value:     price,
			Timestamp: &timestamp,
		})
	}
	return feedData, nil
}

func TickerResponseToFeedData(data TickerResponse, feedMap map[string]int32) (*common.FeedData, error) {
	loc, _ := time.LoadLocation("Asia/Seoul")

	date, err := time.ParseInLocation(dateLayout, data.Content.Date, loc)
	if err != nil {
		log.Error().Err(err).Msg("error in bithumb.TickerResponseToFeedData")
		return nil, err
	}

	t, err := time.ParseInLocation(timeLayout, data.Content.Time, loc)
	if err != nil {
		log.Error().Err(err).Msg("error in bithumb.TickerResponseToFeedData")
		return nil, err
	}

	timestamp := time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	timestamp = timestamp.UTC()

	price, err := common.PriceStringToFloat64(data.Content.ClosePrice)
	if err != nil {
		log.Error().Err(err).Msg("error in bithumb.TickerResponseToFeedData")
		return nil, err
	}
	splitted := strings.Split(data.Content.Symbol, "_")
	symbol := splitted[0] + "-" + splitted[1]

	id, exists := feedMap[symbol]
	if !exists {
		log.Warn().Str("Player", "bithumb").Str("symbol", symbol).Msg("feed not found")
		return nil, nil
	}

	return &common.FeedData{
		FeedID:    id,
		Value:     price,
		Timestamp: &timestamp,
	}, nil

}
