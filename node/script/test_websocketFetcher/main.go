package main

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/bybit"
	"github.com/rs/zerolog/log"
)

func main() {

	ctx := context.Background()
	feed := []common.Feed{
		{
			ID:         1,
			Name:       "bybit-wss-BTC-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "bybit", "base": "btc", "quote": "usdt"}`),
			ConfigID:   1,
		},
		{
			ID:         2,
			Name:       "bybit-wss-ETH-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "bybit", "base": "eth", "quote": "usdt"}`),
			ConfigID:   2,
		},
		{
			ID:         3,
			Name:       "bybit-wss-ADA-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "bybit", "base": "ada", "quote": "usdt"}`),
			ConfigID:   3,
		},
		{
			ID:         4,
			Name:       "bybit-wss-BNB-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "bybit", "base": "bnb", "quote": "usdt"}`),
			ConfigID:   4,
		},
	}
	feedMap := common.GetWssFeedMap(feed)

	ch := make(chan *common.FeedData)
	fetcher, err := bybit.New(ctx, common.WithFeedDataBuffer(ch), common.WithFeedMaps(feedMap["bybit"]))
	if err != nil {
		log.Error().Err(err).Msg("failed to create bybit fetcher")
		return
	}

	go fetcher.Run(ctx)
	for {
		select {
		case data := <-ch:
			fmt.Println(data)
		case <-ctx.Done():
			return
		}
	}
}
