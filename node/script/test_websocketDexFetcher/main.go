package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/uniswap"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	feeds := []common.Feed{
		{
			ID:   1,
			Name: "Dragonswap-ZP-KAIA",
			Definition: json.RawMessage(`{
				"chainId": "8217",
				"address": "0xb1bcb17975407f2e9dbae748366b95e12f163a03",
				"type": "UniswapPool",
				"token0Decimals": 18,
				"token1Decimals": 18,
				"reciprocal": true
				}`),
			ConfigID: 1,
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
	fetcher := uniswap.New(common.WithDexFeedDataBuffer(ch), common.WithWebsocketChainReader(chainReader), common.WithFeeds(feeds))
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
