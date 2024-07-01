package main

import (
	"context"
	"encoding/json"
	"os"

	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/websocketfetcher/providers/uniswap"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	feeds := []common.Feed{
		{
			ID:   1,
			Name: "UniswapV3-0.3-WBTC-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x9db9e0e53058c89e5b94e29621a205198648425b",
				"type": "UniswapPool",
				"token0Decimals": 8,
				"token1Decimals": 6
				}`),
			ConfigID: 1,
		},
		{
			ID:   2,
			Name: "UniswapV3-DAI-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x48da0965ab2d2cbf1c17c09cfb5cbe67ad5b1406",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 6
				}`),
			ConfigID: 2,
		},
		{
			ID:   3,
			Name: "UniswapV3:0.3-ETH-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x4e68ccd3e89f51c3074ca5072bbac773960dfa36",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 6
			}`),
			ConfigID: 3,
		},
		{
			ID:   4,
			Name: "UniswapV3:0.05-ETH-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x11b815efb8f581194ae79006d24e0d814b7697f6",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 6
			}`),
			ConfigID: 4,
		},
		{
			ID:   5,
			Name: "UniswapV3-UNI-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x3470447f3cecffac709d3e783a307790b0208d60",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 6
			}`),
			ConfigID: 5,
		},
		{
			ID:   6,
			Name: "KlaySwap-PER-KLAY",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0x45ccd8a73053ab94efb7a9d4fd48da888c2977f3",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 18
			}`),
			ConfigID: 6,
		},
		{
			ID:   7,
			Name: "UniswapV3-0.01-USDC-USDT",
			Definition: json.RawMessage(`{
				"chainId": "1",
				"address": "0x3416cf6c708da44db2624d63ea0aaef7113527c6",
				"type": "UniswapPool",
				"token0Decimals": 6,
				"token1Decimals": 6
			}`),
			ConfigID: 7,
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

	ch := make(chan common.FeedData)
	fetcher := uniswap.New(common.WithDexFeedDataBuffer(ch), common.WithWebsocketChainReader(chainReader), common.WithFeeds(feeds))
	fetcher.Run(ctx)
	for {
		select {
		case data := <-ch:
			log.Info().Interface("data", data).Msg("data")
		case <-ctx.Done():

			return
		}
	}

}
