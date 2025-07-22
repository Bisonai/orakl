package main

import (
	"context"
	"encoding/json"
	"fmt"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/gateio"
	"github.com/rs/zerolog/log"
)

func main() {

	ctx := context.Background()
	feed := []common.Feed{
		{
			ID:         1,
			Name:       "gateio-wss-BTC-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "gateio", "base": "btc", "quote": "usdt"}`),
			ConfigID:   1,
		},
		{
			ID:         2,
			Name:       "gateio-wss-ETH-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "gateio", "base": "eth", "quote": "usdt"}`),
			ConfigID:   2,
		},
		{
			ID:         3,
			Name:       "gateio-wss-ADA-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "gateio", "base": "ada", "quote": "usdt"}`),
			ConfigID:   3,
		},
		{
			ID:         4,
			Name:       "gateio-wss-BNB-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "gateio", "base": "bnb", "quote": "usdt"}`),
			ConfigID:   4,
		},
	}
	feedMap := common.GetWssFeedMap(feed)

	ch := make(chan *common.FeedData)
	fetcher, err := gateio.New(ctx, common.WithFeedDataBuffer(ch), common.WithFeedMaps(feedMap["gateio"]))
	if err != nil {
		log.Error().Err(err).Msg("failed to create gateio fetcher")
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
