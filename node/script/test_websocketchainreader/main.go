package main

import (
	"context"
	"os"

	"bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"
	"github.com/klaytn/klaytn/blockchain/types"

	"github.com/rs/zerolog/log"
)

// uniswapV3 pool event
const Event = `event Swap(
    address indexed sender,
    address indexed recipient,
    int256 amount0,
    int256 amount1,
    uint160 sqrtPriceX96,
    uint128 liquidity,
    int24 tick
  )`

func main() {
	kaiaWebsocketUrl := os.Getenv("KAIA_WEBSOCKET_URL")
	ethWebsocketUrl := os.Getenv("ETH_WEBSOCKET_URL")

	ctx := context.Background()
	chainReader, err := websocketchainreader.New(kaiaWebsocketUrl, ethWebsocketUrl)
	if err != nil {
		log.Error().Err(err).Msg("failed to create websocketchainreader")
		return
	}

	channel := make(chan types.Log)
	address := "0x11b815efb8f581194ae79006d24e0d814b7697f6" // ETH-USDT 0.03

	eventName, input, _, err := utils.ParseMethodSignature(Event)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse method signature")
		return
	}

	abi, err := utils.GenerateEventABI(eventName, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate event abi")
		return
	}

	err = chainReader.Subscribe(ctx, websocketchainreader.WithAddress(address), websocketchainreader.WithChannel(channel))
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe")
		return
	}

	for eventLog := range channel {
		res, err := abi.Unpack(eventName, eventLog.Data)
		if err != nil {
			continue
		}
		log.Info().Interface("event", res).Msg("event")
		// sqrtprice := res[2]
		// price, err := getTokenPrice(sqrtprice.(*big.Int))
	}
}
