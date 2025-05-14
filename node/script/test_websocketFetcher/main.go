package main

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/btse"
	"github.com/rs/zerolog/log"
)

func main() {

	ctx := context.Background()
	feed := []common.Feed{
		{
			ID:         1,
			Name:       "btse-wss-BTC-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "btse", "base": "btc", "quote": "usdt"}`),
			ConfigID:   1,
		},
		{
			ID:         2,
			Name:       "btse-wss-ETH-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "btse", "base": "eth", "quote": "usdt"}`),
			ConfigID:   2,
		},
		{
			ID:         3,
			Name:       "btse-wss-ADA-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "btse", "base": "ada", "quote": "usdt"}`),
			ConfigID:   3,
		},
		{
			ID:         4,
			Name:       "btse-wss-BNB-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "btse", "base": "bnb", "quote": "usdt"}`),
			ConfigID:   4,
		},
	}
	feedMap := common.GetWssFeedMap(feed)

	ch := make(chan *common.FeedData)
	fetcher, err := btse.New(ctx, common.WithFeedDataBuffer(ch), common.WithFeedMaps(feedMap["btse"]))
	if err != nil {
		log.Error().Err(err).Msg("failed to create btse fetcher")
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
