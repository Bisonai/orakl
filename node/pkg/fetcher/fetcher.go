package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

func (f *Fetcher) Run(ctx context.Context, chainHelpers map[string]ChainHelper, proxies []Proxy) {
	fetcherCtx, cancel := context.WithCancel(ctx)
	f.fetcherCtx = fetcherCtx
	f.cancel = cancel
	f.isRunning = true

	ticker := time.NewTicker(FETCHER_FREQUENCY)
	go func() {
		for {
			select {
			case <-f.fetcherCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := f.fetchAndInsert(f.fetcherCtx, chainHelpers, proxies)
				if err != nil {
					log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetchAndInsert")
				}
			}
		}
	}()
}

func (f *Fetcher) fetchAndInsert(ctx context.Context, chainHelpers map[string]ChainHelper, proxies []Proxy) error {
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("fetching and inserting")
	results, err := f.fetch(chainHelpers, proxies)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("fetch complete")

	err = insertFeedData(ctx, f.Adapter.ID, results)
	if err != nil {
		return err
	}

	rawValues := make([]float64, len(results))
	for i, result := range results {
		rawValues[i] = result.Value
	}

	aggregated, err := calculator.GetFloatMed(rawValues)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Float64("aggregated", aggregated).Msg("aggregated")

	err = insertPgsql(ctx, f.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("inserted into pgsql")

	err = insertRdb(ctx, f.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("inserted into rdb")
	return nil
}

func (f *Fetcher) fetch(chainHelpers map[string]ChainHelper, proxies []Proxy) ([]FeedData, error) {
	feeds := f.Feeds

	data := []FeedData{}
	dataChan := make(chan FeedData)
	errChan := make(chan error)

	defer close(dataChan)
	defer close(errChan)

	for _, feed := range feeds {
		go func(feed Feed) {
			definition := new(Definition)
			err := json.Unmarshal(feed.Definition, &definition)
			if err != nil {
				errChan <- err
				return
			}

			var resultValue float64
			var fetchErr error

			switch {
			case definition.Type == nil:
				resultValue, fetchErr = f.cex(definition, proxies)
				if fetchErr != nil {
					errChan <- fetchErr
					return
				}
			case *definition.Type == "UniswapPool":
				resultValue, fetchErr = f.uniswapV3(definition, chainHelpers)
				if fetchErr != nil {
					errChan <- fetchErr
					return
				}
			default:
				errChan <- errors.New("unknown fetcher type")
			}

			dataChan <- FeedData{FeedName: feed.Name, Value: resultValue}

		}(feed)
	}

	for i := 0; i < len(feeds); i++ {
		select {
		case result := <-dataChan:
			data = append(data, result)
		case err := <-errChan:
			log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetch")
		}
	}

	if len(data) < 1 {
		return nil, errors.New("no data fetched")
	}
	return data, nil
}

func (f *Fetcher) cex(definition *Definition, proxies []Proxy) (float64, error) {
	rawResult, err := f.requestFeed(definition, proxies)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in requestFeed")
		return 0, err
	}

	return reducer.Reduce(rawResult, definition.Reducers)
}

func (f *Fetcher) uniswapV3(definition *Definition, chainHelpers map[string]ChainHelper) (float64, error) {
	if definition.Address == nil || definition.ChainID == nil || definition.Token0Decimals == nil || definition.Token1Decimals == nil {
		log.Error().Any("definition", definition).Msg("missing required fields for uniswapV3")
		return 0, errors.New("missing required fields for uniswapV3")
	}

	helper := chainHelpers[*definition.ChainID]
	if helper == nil {
		return 0, errors.New("chain helper not found")
	}

	rawResult, err := helper.ReadContract(context.Background(), *definition.Address, "function slot0() external view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)")
	if err != nil {
		log.Error().Err(err).Msg("failed to read contract for uniswap v3 pool contract")
		return 0, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok || len(rawResultSlice) < 1 {
		return 0, errors.New("unexpected raw result type")
	}

	sqrtPriceX96, ok := rawResultSlice[0].(*big.Int)
	if !ok {
		return 0, errors.New("unexpected result on converting to bigint")
	}

	result := getTokenPrice(sqrtPriceX96, int(*definition.Token0Decimals), int(*definition.Token1Decimals))
	return result, nil
}

func (f *Fetcher) requestFeed(definition *Definition, proxies []Proxy) (interface{}, error) {
	var filteredProxy []Proxy
	if definition.Location != nil && *definition.Location != "" {
		filteredProxy = f.filterProxyByLocation(proxies, *definition.Location)
	} else {
		filteredProxy = proxies
	}

	if len(filteredProxy) > 0 {
		proxy := filteredProxy[rand.Intn(len(filteredProxy))]
		proxyUrl := fmt.Sprintf("%s://%s:%d", proxy.Protocol, proxy.Host, proxy.Port)
		log.Debug().Str("Player", "Fetcher").Str("proxyUrl", proxyUrl).Msg("using proxy")
		return f.requestWithProxy(definition, proxyUrl)
	}

	return f.requestWithoutProxy(definition)
}

func (f *Fetcher) requestWithoutProxy(definition *Definition) (interface{}, error) {
	return request.GetRequest[interface{}](*definition.Url, nil, definition.Headers)
}

func (f *Fetcher) requestWithProxy(definition *Definition, proxyUrl string) (interface{}, error) {
	return request.GetRequestProxy[interface{}](*definition.Url, nil, definition.Headers, proxyUrl)
}

func (f *Fetcher) filterProxyByLocation(proxies []Proxy, location string) []Proxy {
	filteredProxies := []Proxy{}
	for _, proxy := range proxies {
		if proxy.Location != nil && *proxy.Location == location {
			filteredProxies = append(proxies, proxy)
		}
	}
	return filteredProxies
}

func getTokenPrice(sqrtPriceX96 *big.Int, decimal0 int, decimal1 int) float64 {
	sqrtPriceX96Float := new(big.Float).SetInt(sqrtPriceX96)
	sqrtPriceX96Float.Quo(sqrtPriceX96Float, new(big.Float).SetFloat64(math.Pow(2, 96)))
	sqrtPriceX96Float.Mul(sqrtPriceX96Float, sqrtPriceX96Float) // square

	decimalDiff := new(big.Float).SetFloat64(math.Pow(10, float64(decimal1-decimal0)))

	datum := sqrtPriceX96Float.Quo(sqrtPriceX96Float, decimalDiff)

	multiplier := new(big.Float).SetFloat64(math.Pow(10, 6))
	datum.Mul(datum, multiplier)

	result, _ := datum.Float64()

	return math.Round(result)
}

func insertFeedData(ctx context.Context, adapterId int64, feedData []FeedData) error {
	insertRows := make([][]any, 0, len(feedData))
	for _, data := range feedData {
		insertRows = append(insertRows, []any{adapterId, data.FeedName, data.Value})
	}

	_, err := db.BulkInsert(ctx, "feed_data", []string{"adapter_id", "name", "value"}, insertRows)
	return err
}

func insertPgsql(ctx context.Context, name string, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"name": name, "value": int64(value)})
	return err
}

func insertRdb(ctx context.Context, name string, value float64) error {
	key := "localAggregate:" + name
	data, err := json.Marshal(redisAggregate{Value: int64(value), Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}
