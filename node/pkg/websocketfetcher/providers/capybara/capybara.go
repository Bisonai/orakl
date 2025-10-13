package capybara

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/kaiachain/kaia/blockchain/types"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

type CapybaraFetcher common.DexFetcher

const (
	quotePotentialSwap = `function quotePotentialSwap(
    address fromToken,
    address toToken,
    int256 fromAmount
) public view override returns (uint256 potentialOutcome, uint256 haircut)`

	event = `event SwapV2(
    address indexed sender,
    address from_token,
    address to_token,
    uint256 from_amount,
    uint256 to_amount,
    uint256 to_token_fee,
    address indexed recipient
)`
)

func New(opts ...common.DexFetcherOption) common.FetcherInterface {
	config := &common.DexFetcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return &CapybaraFetcher{
		Feeds:                config.Feeds,
		FeedDataBuffer:       config.FeedDataBuffer,
		WebsocketChainReader: config.WebsocketChainReader,
		LatestEntries:        make(map[int32]*common.FeedData),
	}
}

func (f *CapybaraFetcher) Run(ctx context.Context) {
	for _, feed := range f.Feeds {
		go f.run(ctx, feed)

		time.Sleep(1 * time.Second)
	}
}

func (f *CapybaraFetcher) run(ctx context.Context, feed common.Feed) {
	price, err := f.getInitialPrice(ctx, feed)
	if err != nil {
		log.Error().Str("Player", "Capybara").Err(err).Msg("error in capybara.run, failed to get initial price")
		return
	}

	now := time.Now()
	initialFeedData := &common.FeedData{
		FeedID:    feed.ID,
		Value:     *price,
		Timestamp: &now,
	}
	log.Debug().Str("Player", "Capybara").Any("feedData", initialFeedData).Msg("initial price fetched")
	f.FeedDataBuffer <- initialFeedData
	f.Mutex.Lock()
	f.LatestEntries[feed.ID] = initialFeedData
	f.Mutex.Unlock()

	go func(ctx context.Context) {
		t := time.NewTicker(time.Minute)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				f.Mutex.Lock()
				last, ok := f.LatestEntries[feed.ID]
				f.Mutex.Unlock()
				if !ok {
					continue
				}

				if time.Since(*last.Timestamp) > time.Hour {
					price, err := f.getInitialPrice(ctx, feed)
					if err != nil {
						log.Error().Str("Player", "Capybara").Err(err).Msg("failed to get refreshed price")
						continue
					}

					now := time.Now()
					refreshedFeedData := &common.FeedData{
						FeedID:    feed.ID,
						Value:     *price,
						Timestamp: &now,
					}
					f.FeedDataBuffer <- refreshedFeedData

					f.Mutex.Lock()
					f.LatestEntries[feed.ID] = refreshedFeedData
					f.Mutex.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	err = f.subscribeEvent(ctx, feed)
	if err != nil {
		log.Error().Str("Player", "Capybara").Err(err).Msg("error in capybara.run, failed to subscribe event")
		return
	}
}

func (f *CapybaraFetcher) getInitialPrice(ctx context.Context, feed common.Feed) (*float64, error) {
	definition := new(common.DexFeedDefinitionCapybara)
	err := json.Unmarshal(feed.Definition, &definition)
	if err != nil {
		log.Error().Str("Player", "Capybara").Err(err).Msg("error in capybara.getInitialPrice, failed to unmarshal definition")
		return nil, err
	}

	return f.getPriceThroughQuotePotentialSwap(ctx, definition)
}

func (f *CapybaraFetcher) getPriceThroughQuotePotentialSwap(ctx context.Context, definition *common.DexFeedDefinitionCapybara) (*float64, error) {
	var initAmount int64 = 10 // ex. 10 klay -> usdt or 10 weth -> usdt
	if definition.InitAmount > 0 {
		initAmount = definition.InitAmount
	}

	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "Capybara").Str("chainId", definition.ChainId).Msg("error in capybara.getPriceThroughQuotePotentialSwap, chain type not found")
		return nil, errorSentinel.ErrFetcherNoMatchingChainID
	}

	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(definition.Token0Decimals)), nil)
	defaultAmount := new(big.Int).Mul(decimals, big.NewInt(initAmount))
	rawResult, err := f.WebsocketChainReader.ReadContractOnce(ctx, chainType, definition.Address, quotePotentialSwap, kaiacommon.HexToAddress(definition.Token0Address), kaiacommon.HexToAddress(definition.Token1Address), defaultAmount)
	if err != nil {
		log.Error().Str("Player", "Capybara").Err(err).Msg("error in capybara.getPriceThroughQuotePotentialSwap, failed to read contract")
		return nil, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok || len(rawResultSlice) < 1 {
		log.Error().Str("Player", "Capybara").Msg("error in capybara.getPriceThroughQuotePotentialSwap, failed to get slice result")
		return nil, errorSentinel.ErrFetcherFailedToGetDexResultSlice
	}

	rawPrice, ok := rawResultSlice[0].(*big.Int)
	if !ok {
		log.Error().Str("Player", "Capybara").Msg("error in capybara.getPriceThroughQuotePotentialSwap, failed to convert raw price")
		return nil, errorSentinel.ErrFetcherFailedBigIntConvert
	}

	haircut, ok := rawResultSlice[1].(*big.Int)
	if !ok {
		log.Error().Str("Player", "Capybara").Msg("error in capybara.getPriceThroughQuotePotentialSwap, failed to convert raw haircut")
		return nil, errorSentinel.ErrFetcherFailedBigIntConvert
	}

	rawPrice = rawPrice.Add(rawPrice, haircut)
	rawPrice = rawPrice.Div(rawPrice, big.NewInt(initAmount))

	return rawPriceToFloat64(rawPrice, definition.Token1Decimals)
}

func (f *CapybaraFetcher) subscribeEvent(ctx context.Context, feed common.Feed) error {
	definition := new(common.DexFeedDefinitionCapybara)
	err := json.Unmarshal(feed.Definition, &definition)
	if err != nil {
		log.Error().Str("Player", "Capybara").Err(err).Msg("error in capybara.subscribeEvent, failed to unmarshal definition")
		return err
	}

	return f.readSwapEvent(ctx, feed, definition)
}

func (f *CapybaraFetcher) readSwapEvent(ctx context.Context, feed common.Feed, definition *common.DexFeedDefinitionCapybara) error {
	logChannel := make(chan types.Log)
	address := definition.Address

	chainType, ok := f.WebsocketChainReader.ChainIdToChainType[definition.ChainId]
	if !ok {
		log.Error().Str("Player", "Capybara").Str("chainId", definition.ChainId).Msg("error in capybara.getInitialPrice, chain type not found")
		return errorSentinel.ErrFetcherNoMatchingChainID
	}

	var eventName, input string
	var eventParseErr error

	eventName, input, _, eventParseErr = utils.ParseMethodSignature(event)
	if eventParseErr != nil {
		return eventParseErr
	}

	swapEventABI, err := utils.GenerateEventABI(eventName, input)
	if err != nil {
		return err
	}

	err = f.WebsocketChainReader.Subscribe(
		ctx,
		websocketchainreader.WithAddress(address),
		websocketchainreader.WithChannel(logChannel),
		websocketchainreader.WithChainType(chainType))
	if err != nil {
		return err
	}
	decimalsFactor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(definition.Token0Decimals)), nil)

	for eventLog := range logChannel {
		res, err := swapEventABI.Unpack(eventName, eventLog.Data)
		if err != nil {
			continue
		}

		fromToken := res[0].(kaiacommon.Address)
		toToken := res[1].(kaiacommon.Address)

		if !(strings.EqualFold(fromToken.Hex(), definition.Token0Address) &&
			strings.EqualFold(toToken.Hex(), definition.Token1Address)) &&
			!(strings.EqualFold(fromToken.Hex(), definition.Token1Address) &&
				strings.EqualFold(toToken.Hex(), definition.Token0Address)) {
			continue
		}

		fromAmount := res[2].(*big.Int)
		toAmount := res[3].(*big.Int)

		if fromAmount == nil || toAmount == nil {
			continue
		}

		var rawPrice *big.Int
		if strings.EqualFold(fromToken.String(), definition.Token0Address) {
			rawPrice = toAmount.Mul(toAmount, decimalsFactor)
			rawPrice = rawPrice.Div(rawPrice, fromAmount)
		} else {
			rawPrice = fromAmount.Mul(fromAmount, decimalsFactor)
			rawPrice = rawPrice.Div(rawPrice, toAmount)
		}

		price, err := rawPriceToFloat64(rawPrice, definition.Token1Decimals)
		if err != nil {
			continue
		}
		now := time.Now()
		feedData := &common.FeedData{
			FeedID:    feed.ID,
			Value:     *price,
			Timestamp: &now,
		}
		log.Debug().Str("Player", "Capybara").Any("feedData", feedData).Msg("price fetched")
		f.FeedDataBuffer <- feedData
		f.Mutex.Lock()
		f.LatestEntries[feed.ID] = feedData
		f.Mutex.Unlock()
	}

	return nil
}

func rawPriceToFloat64(rawPrice *big.Int, toTokenDecimals int) (*float64, error) {
	if rawPrice == nil {
		return nil, errorSentinel.ErrFetcherInvalidInput
	}

	if toTokenDecimals < 0 {
		return nil, errorSentinel.ErrFetcherInvalidInput
	}

	price := new(big.Float).SetInt(rawPrice)
	denom := new(big.Float).SetFloat64(math.Pow(10, float64(toTokenDecimals)))

	price.Quo(price, denom)

	price.Mul(price, new(big.Float).SetFloat64(math.Pow(10, common.DECIMALS)))

	result, _ := price.Float64()
	result = math.Round(result)

	return &result, nil
}
