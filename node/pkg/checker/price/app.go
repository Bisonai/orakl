package price

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/utils/request"

	"bisonai.com/miko/node/pkg/alert"
	"github.com/rs/zerolog/log"
)

const (
	defaultTimeout            = 5 * time.Second
	dalEndpoint               = "https://dal.cypress.orakl.network"
	binanceEndpoint           = "https://api.binance.com/api/v3/ticker/price"
	pythEndpoint              = "https://hermes.pyth.network"
	defaultPriceDiffThreshold = 0.01 // 1%
	checkInterval             = 15 * time.Second
)

func dalSymbolToBaseAndQuote(symbol string) (baseAndQuote, error) {
	splitted := strings.Split(symbol, "-")
	if len(splitted) != 2 {
		return baseAndQuote{}, errors.New("invalid dal symbol element length")
	}
	return baseAndQuote{base: splitted[0], quote: splitted[1]}, nil
}

func Start(ctx context.Context, opts ...Option) error {
	app := &App{
		trackingPairs:      map[baseAndQuote]struct{}{},
		priceDiffThreshold: defaultPriceDiffThreshold,
	}

	for _, opt := range opts {
		opt(app)
	}

	if len(app.trackingPairs) == 0 {
		return errors.New("no tracking pairs provided")
	}

	app.run(ctx)
	return nil
}

func (a *App) run(ctx context.Context) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.process(); err != nil {
				log.Error().Err(err).Msg("failed to run once")
			}
		}
	}
}

func (a *App) process() error {
	var (
		dalPrices, binancePrices, pythPrices                       map[baseAndQuote]float64
		dalPriceFetchErr, binancePriceFetchErr, pythPricesFetchErr error
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		dalPrices, dalPriceFetchErr = a.collectDalPrices()
		log.Info().Int("len", len(dalPrices)).Msg("fetched dal prices")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		binancePrices, binancePriceFetchErr = a.collectBinancePrices()
		log.Info().Int("len", len(binancePrices)).Msg("fetched binance prices")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pythPrices, pythPricesFetchErr = a.collectPythPrices()
		log.Info().Int("len", len(pythPrices)).Msg("fetched pyth prices")
	}()

	wg.Wait()

	if dalPriceFetchErr != nil {
		return dalPriceFetchErr
	}

	if binancePriceFetchErr != nil {
		return binancePriceFetchErr
	}

	if pythPricesFetchErr != nil {
		return pythPricesFetchErr
	}

	// compare prices
	log.Info().Msg("comparing prices")
	for e, p := range dalPrices {
		if p == 0 {
			log.Info().Float64("price", p).Any("baseAndQuote", e).Msg("price is zero")
			continue
		}

		bp, ok := binancePrices[e]
		if !ok {
			continue
		}
		log.Info().Str("base", e.base).Str("quote", e.quote).Float64("dal_price", p).Float64("binance_price", binancePrices[e]).Msg("price difference")

		if (math.Abs(p-bp) / p) > a.priceDiffThreshold {
			alert.SlackAlertWithEndpoint(fmt.Sprintf("3%% exceeded price difference detected for %s-%s [dal: %f, binance: %f]", e.base, e.quote, p, bp), a.slackEndpoint)
		}

		pp, ok := pythPrices[e]
		if !ok {
			continue
		}
		log.Info().Str("base", e.base).Str("quote", e.quote).Float64("dal_price", p).Float64("pyth_price", pp).Msg("price difference")

		if (math.Abs(p-pp) / p) > a.priceDiffThreshold {
			alert.SlackAlertWithEndpoint(fmt.Sprintf("3%% exceeded price difference detected for %s-%s [dal: %f, pyth: %f]", e.base, e.quote, p, pp), a.slackEndpoint)
		}
	}
	return nil
}

func (a *App) collectDalPrices() (map[baseAndQuote]float64, error) {
	prices := map[baseAndQuote]float64{}

	tickers, err := a.getDalTickers()
	if err != nil {
		return nil, err
	}

	for _, t := range tickers {
		price, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			log.Error().Err(err).Str("symbol", t.Symbol).Msg("failed to parse price")
			continue
		}

		decimals, err := strconv.ParseInt(t.Decimals, 10, 64)
		if err != nil {
			log.Error().Err(err).Str("symbol", t.Symbol).Msg("failed to parse decimals")
			continue
		}

		price = price / math.Pow(10, float64(decimals))

		e, err := dalSymbolToBaseAndQuote(t.Symbol)
		if err != nil {
			log.Error().Err(err).Str("symbol", t.Symbol).Msg("failed to parse dal symbol")
			continue
		}

		if _, ok := a.trackingPairs[e]; !ok {
			continue
		}

		prices[e] = price
	}

	return prices, nil
}

func (a *App) collectBinancePrices() (map[baseAndQuote]float64, error) {
	prices := map[baseAndQuote]float64{}

	tickers, err := a.getBinanceTickers()
	if err != nil {
		return nil, err
	}

	dalSymbolToBinanceSymbolMapping := map[string]baseAndQuote{}
	for b := range a.trackingPairs {
		dalSymbolToBinanceSymbolMapping[b.toBinanceSymbol()] = b
	}

	for _, t := range tickers {
		e, ok := dalSymbolToBinanceSymbolMapping[t.Symbol]
		if !ok {
			continue
		}

		price, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			log.Error().Err(err).Str("symbol", t.Symbol).Msg("failed to parse price")
			continue
		}

		prices[e] = price
	}

	return prices, nil
}

func (a *App) collectPythPrices() (map[baseAndQuote]float64, error) {
	prices := map[baseAndQuote]float64{}

	feedInfos, err := a.getPythFeedInfo()
	if err != nil {
		return nil, err
	}

	idToBaseAndQuote := map[string]baseAndQuote{}

	feedIds := make([]string, 0, len(feedInfos))
	for _, info := range feedInfos {
		base := info.Attributes.Base
		quote := info.Attributes.QuoteCurrency

		if !slices.Contains([]string{"USDT", "USD", "USDC"}, quote) {
			continue
		}

		if _, ok := a.trackingPairs[baseAndQuote{base: base, quote: "USDT"}]; !ok {
			continue
		}

		feedIds = append(feedIds, info.Id)
		idToBaseAndQuote[info.Id] = baseAndQuote{base: base, quote: "USDT"}
	}

	priceResponse, err := a.getPythPrices(feedIds)
	if err != nil {
		return nil, err
	}

	for _, priceEntry := range priceResponse.Parsed {
		baseAndQuote, ok := idToBaseAndQuote[priceEntry.Id]
		if !ok {
			continue
		}

		price, err := strconv.ParseFloat(priceEntry.Price.Price, 64)
		if err != nil {
			log.Error().Err(err).Str("id", priceEntry.Id).Msg("failed to parse price")
			continue
		}

		price = price * math.Pow(10, float64(priceEntry.Price.Expo))

		prices[baseAndQuote] = price
	}

	return prices, nil

}

func (a *App) getBinanceTickers() ([]binanceResponse, error) {
	return request.Request[[]binanceResponse](
		request.WithEndpoint(binanceEndpoint),
		request.WithTimeout(defaultTimeout),
	)
}

func (a *App) getDalTickers() ([]dalResponse, error) {
	return request.Request[[]dalResponse](
		request.WithEndpoint(dalEndpoint+"/latest-data-feeds/all"),
		request.WithTimeout(defaultTimeout),
		request.WithHeaders(map[string]string{"X-API-KEY": a.dalApiKey}),
	)
}

func (a *App) getPythFeedInfo() ([]pythFeedResponse, error) {
	return request.Request[[]pythFeedResponse](
		request.WithEndpoint(pythEndpoint+"/v2/price_feeds?asset_type=crypto"),
		request.WithTimeout(defaultTimeout),
	)
}

func (a *App) getPythPrices(feedIds []string) (pythPriceResponse, error) {
	endpoint := pythEndpoint + "/v2/updates/price/latest?ignore_invalid_price_ids=true"
	for _, feedId := range feedIds {
		endpoint += "&ids[]=" + feedId
	}

	return request.Request[pythPriceResponse](
		request.WithEndpoint(endpoint),
		request.WithTimeout(defaultTimeout),
	)
}
