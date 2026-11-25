package price

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"bisonai.com/orakl/node/pkg/utils/request"
)

// binanceEndpoint = best bid/ask, not last trade: DAL is order-book mid, and last
// trade on coarse-tick pairs (KAIA tick ~0.36%) sits a tick or more off mid.
const (
	defaultTimeout            = 5 * time.Second
	dalEndpoint               = "https://dal.mainnet.orakl.network"
	binanceEndpoint           = "https://api.binance.com/api/v3/ticker/bookTicker"
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

	thresholdGauge.Set(app.priceDiffThreshold)
	// Alerts: Prometheus rules on the exported orakl_price_check_* metrics.
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
		errorsCounter.WithLabelValues("dal").Inc()
	}
	if binancePriceFetchErr != nil {
		errorsCounter.WithLabelValues("binance").Inc()
	}
	if dalPriceFetchErr != nil {
		return dalPriceFetchErr
	}
	if binancePriceFetchErr != nil {
		return binancePriceFetchErr
	}

	log.Info().Msg("comparing prices")
	recordPrices(comparePrices(dalPrices, binancePrices), time.Now())
	return nil
}

// comparePrices pairs DAL and Binance prices ([dal, binance]) of pairs both
// sources returned; zero DAL prices are skipped (diff undefined).
func comparePrices(dalPrices, binancePrices map[baseAndQuote]float64) map[baseAndQuote][2]float64 {
	compared := map[baseAndQuote][2]float64{}
	for e, p := range dalPrices {
		if p == 0 {
			log.Info().Float64("price", p).Any("baseAndQuote", e).Msg("price is zero")
			continue
		}

		bp, ok := binancePrices[e]
		if !ok {
			continue
		}
		log.Info().Str("base", e.base).Str("quote", e.quote).Float64("dal_price", p).Float64("binance_price", bp).Msg("price difference")
		compared[e] = [2]float64{p, bp}
	}
	return compared
}

func diffRatio(dal, binance float64) float64 {
	return math.Abs(dal-binance) / dal
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

		price, err := binanceMid(t)
		if err != nil {
			log.Error().Err(err).Str("symbol", t.Symbol).Msg("failed to parse book ticker")
			continue
		}

		prices[e] = price
	}

	return prices, nil
}

// binanceMid returns (bid+ask)/2; errors on unparsable or empty sides.
func binanceMid(t binanceResponse) (float64, error) {
	bid, err := strconv.ParseFloat(t.BidPrice, 64)
	if err != nil {
		return 0, err
	}
	ask, err := strconv.ParseFloat(t.AskPrice, 64)
	if err != nil {
		return 0, err
	}
	if bid <= 0 || ask <= 0 {
		return 0, errors.New("empty book side")
	}
	return (bid + ask) / 2, nil
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
