package price

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/utils/request"

	"bisonai.com/miko/node/pkg/alert"
	"github.com/rs/zerolog/log"
)

const (
	defaultTimeout     = 5 * time.Second
	dalEndpoint        = "https://dal.cypress.orakl.network"
	binanceEndpoint    = "https://api.binance.com/api/v3/ticker/price"
	priceDiffThreshold = 0.03 // 3%
	checkInterval      = 15 * time.Second
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
		trackingPairs: map[baseAndQuote]struct{}{},
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
		dalPrices, binancePrices               map[baseAndQuote]float64
		dalPriceFetchErr, binancePriceFetchErr error
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
	wg.Wait()

	if dalPriceFetchErr != nil {
		return dalPriceFetchErr
	}

	if binancePriceFetchErr != nil {
		return binancePriceFetchErr
	}

	// compare prices
	log.Info().Msg("comparing prices")
	for e, p := range dalPrices {
		bp, ok := binancePrices[e]
		if !ok {
			continue
		}
		log.Info().Str("base", e.base).Str("quote", e.quote).Float64("dal_price", p).Float64("binance_price", binancePrices[e]).Msg("price difference")

		if (math.Abs(p-bp) / p) > priceDiffThreshold {
			alert.SlackAlertWithEndpoint(fmt.Sprintf("3%% exceeded price difference detected for %s-%s [dal: %f, binance: %f]", e.base, e.quote, p, bp), a.slackEndpoint)
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
