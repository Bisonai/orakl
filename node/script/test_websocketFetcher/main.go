package main

import (
	"context"
	"encoding/json"
	"sync"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/coinbase"
	"github.com/rs/zerolog/log"
)

func main() {
	var wg sync.WaitGroup
	ctx := context.Background()
	feed := []common.Feed{
		{
			ID:         1,
			Name:       "coinbase-wss-BTC-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "coinbase", "base": "btc", "quote": "usdt"}`),
			ConfigID:   1,
		},
		{
			ID:         2,
			Name:       "coinbase-wss-ETH-USDT",
			Definition: json.RawMessage(`{"type": "wss", "provider": "coinbase", "base": "eth", "quote": "usdt"}`),
			ConfigID:   2,
		},
	}
	feedMap := common.GetWssFeedMap(feed)
	fetcher, err := coinbase.New(ctx, common.WithFeedMaps(feedMap["coinbase"]))
	if err != nil {
		log.Error().Err(err).Msg("failed to create coinbase fetcher")
		return
	}
	wg.Add(1)
	fetcher.Run(ctx)

	wg.Wait()
}
