package main

import (
	"context"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/wfetcher"
	"bisonai.com/orakl/node/pkg/wfetcher/common"
	"github.com/rs/zerolog/log"
)

func main() {

	ctx := context.Background()
	feeds := []common.Feed{
		{
			ID:         1,
			Name:       "binance-wss-BTC-USDT",
			Definition: nil,
			ConfigID:   1,
		},
		{
			ID:         2,
			Name:       "coinbase-wss-ADA-USDT",
			Definition: nil,
			ConfigID:   2,
		},
		{
			ID:         3,
			Name:       "coinone-wss-BTC-KRW",
			Definition: nil,
			ConfigID:   3,
		},
		{
			ID:         4,
			Name:       "korbit-wss-BORA-KRW",
			Definition: nil,
			ConfigID:   4,
		},
	}

	app := wfetcher.New()
	err := app.Init(
		ctx,
		wfetcher.WithFeeds(feeds),
		wfetcher.WithBufferSize(100),
		wfetcher.WithStoreInterval(500*time.Millisecond),
	)
	if err != nil {
		log.Error().Err(err).Msg("error in Init")
		return
	}
	go app.Start(ctx)

	rdbCheckInterval := 1000 * time.Millisecond
	ticker := time.NewTicker(rdbCheckInterval)

	feedIds := []int32{1}

	latestKeys := make([]string, len(feedIds))
	for i, feedId := range feedIds {
		latestKeys[i] = "latestFeedData:" + strconv.Itoa(int(feedId))
	}

	for range ticker.C {
		feedData, err := db.MGetObject[common.FeedData](ctx, latestKeys)
		if err != nil {
			log.Error().Err(err).Msg("error in MGetObject")
			continue
		}
		for _, data := range feedData {
			log.Info().Any("FeedData", data).Msg("FeedData")
		}

		bufferData, err := db.PopAllObject[common.FeedData](ctx, "feedDataBuffer")
		if err != nil {
			log.Error().Err(err).Msg("error in PopAllObject")
			continue
		}
		for _, data := range bufferData {
			log.Info().Any("FeedData", data).Msg("FeedDataBuffer")
		}
	}

}
