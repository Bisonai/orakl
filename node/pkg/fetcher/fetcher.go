package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

func NewFetcher(config Config, feeds []Feed) *Fetcher {
	return &Fetcher{
		Config:     config,
		Feeds:      feeds,
		fetcherCtx: nil,
		cancel:     nil,
	}
}

func (f *Fetcher) Run(ctx context.Context, chainHelpers map[string]ChainHelper, proxies []Proxy) {
	fetcherCtx, cancel := context.WithCancel(ctx)
	f.fetcherCtx = fetcherCtx
	f.cancel = cancel
	f.isRunning = true

	fetcherFrequency := time.Duration(f.FetchInterval) * time.Millisecond
	ticker := time.NewTicker(fetcherFrequency)
	go func() {
		for {
			select {
			case <-f.fetcherCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := f.fetcherJob(f.fetcherCtx, chainHelpers, proxies)
				if err != nil {
					log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetchAndInsert")
				}
			}
		}
	}()
}

func (f *Fetcher) fetcherJob(ctx context.Context, chainHelpers map[string]ChainHelper, proxies []Proxy) error {
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("fetcherJob")
	result, err := f.fetch(chainHelpers, proxies)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetch")
		return err
	}

	err = setLatestFeedData(ctx, result, time.Duration(f.FetchInterval)*time.Millisecond)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in setLatestFeedData")
		return err
	}

	return setFeedDataBuffer(ctx, result)
}

func (f *Fetcher) fetch(chainHelpers map[string]ChainHelper, proxies []Proxy) ([]FeedData, error) {
	feeds := f.Feeds

	data := []FeedData{}
	errList := []error{}
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
				errChan <- errorSentinel.ErrFetcherInvalidType
				return
			}
			now := time.Now()
			dataChan <- FeedData{FeedID: feed.ID, Value: resultValue, Timestamp: &now}

		}(feed)
	}

	for i := 0; i < len(feeds); i++ {
		select {
		case result := <-dataChan:
			data = append(data, result)
		case err := <-errChan:
			errList = append(errList, err)
		}
	}

	if len(data) < 1 {
		return nil, errorSentinel.ErrFetcherNoDataFetched
	}

	errString := ""
	if len(errList) > 0 {
		for _, err := range errList {
			errString += err.Error() + "\n"
		}

		log.Warn().Str("Player", "Fetcher").Str("errs", errString).Msg("errors in fetching")
	}

	return data, nil
}

func (f *Fetcher) cex(definition *Definition, proxies []Proxy) (float64, error) {
	rawResult, err := f.requestFeed(definition, proxies)
	if err != nil {
		log.Warn().Str("Player", "Fetcher").Err(err).Msg("error in requestFeed")
		return 0, err
	}

	return reducer.Reduce(rawResult, definition.Reducers)
}

func (f *Fetcher) uniswapV3(definition *Definition, chainHelpers map[string]ChainHelper) (float64, error) {
	if definition.Address == nil || definition.ChainID == nil || definition.Token0Decimals == nil || definition.Token1Decimals == nil {
		log.Error().Any("definition", definition).Msg("missing required fields for uniswapV3")
		return 0, errorSentinel.ErrFetcherInvalidDexFetcherDefinition
	}

	helper := chainHelpers[*definition.ChainID]
	if helper == nil {
		return 0, errorSentinel.ErrFetcherChainHelperNotFound
	}

	rawResult, err := helper.ReadContract(context.Background(), *definition.Address, "function slot0() external view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)")
	if err != nil {
		log.Error().Err(err).Msg("failed to read contract for uniswap v3 pool contract")
		return 0, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok || len(rawResultSlice) < 1 {
		return 0, errorSentinel.ErrFetcherInvalidRawResult
	}

	sqrtPriceX96, ok := rawResultSlice[0].(*big.Int)
	if !ok {
		return 0, errorSentinel.ErrFetcherConvertToBigInt
	}

	return getTokenPrice(sqrtPriceX96, definition)
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
			filteredProxies = append(filteredProxies, proxy)
		}
	}
	return filteredProxies
}
