package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/capybara"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	feeds := []common.Feed{
		{
			ID:   1,
			Name: "Capybara-WALK-USDT",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0x872e7e7422bcacdcb37f7fffb0cfe3f2f0d6c546",
				"type": "CapybaraPool",
				"token0Decimals": 18,
				"token1Decimals": 6,
				"token0Address": "0x976232eb7eb92287ff06c5d145bd0d1c033eca58",
				"token1Address": "0x5c13e303a62fc5dedf5b52d66873f2e59fedadc2"
				}`),
			ConfigID: 1,
		},
		{
			ID:   2,
			Name: "Capybara-KRWO-USDT",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0x4b63ec6284810f62cecba6f03cf17413b0f4cec3",
				"type": "CapybaraPool",
				"token0Decimals": 6,
				"token1Decimals": 6,
				"token0Address": "0x7fc692699f2216647a0e06225d8bdf8cdee40e7f",
				"token1Address": "0x5c13e303a62fc5dedf5b52d66873f2e59fedadc2"
			}`),
			ConfigID: 2,
		},
		{
			ID:   3,
			Name: "Capybara-KAIA-USDT(Wormhole)",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0x6389dbfa1427a3b0a89cddc7ea9bbda6e73dece7",
				"type": "CapybaraPool",
				"token0Decimals": 18,
				"token1Decimals": 6,
				"token0Address": "0x19aac5f612f524b754ca7e7c41cbfa2e981a4432",
				"token1Address": "0x5c13e303a62fc5dedf5b52d66873f2e59fedadc2"
			}`),
			ConfigID: 3,
		},
		{
			ID:   4,
			Name: "Capybara-KAIA-USDT(Stargate)",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0x1de1578476d9b4237f963eca5d37500fc33df3d1",
				"type": "CapybaraPool",
				"token0Decimals": 18,
				"token1Decimals": 6,
				"token0Address": "0x19aac5f612f524b754ca7e7c41cbfa2e981a4432",
				"token1Address": "0x9025095263d1e548dc890a7589a4c78038ac40ab"
			}`),
			ConfigID: 4,
		},
	}

	kaiaWebsocketUrl := os.Getenv("KAIA_WEBSOCKET_URL")
	ethWebsocketUrl := os.Getenv("ETH_WEBSOCKET_URL")

	chainReader, err := websocketchainreader.New(
		websocketchainreader.WithEthWebsocketUrl(ethWebsocketUrl),
		websocketchainreader.WithKaiaWebsocketUrl(kaiaWebsocketUrl),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create websocketchainreader")
		return
	}

	ch := make(chan *common.FeedData)
	fetcher := capybara.New(common.WithDexFeedDataBuffer(ch), common.WithWebsocketChainReader(chainReader), common.WithFeeds(feeds))
	fetcher.Run(ctx)
	for {
		select {
		case data := <-ch:
			log.Info().Interface("data", data).Str("price", fmt.Sprintf("%f", data.Value)).Msg("data")
		case <-ctx.Done():
			return
		}
	}

}
