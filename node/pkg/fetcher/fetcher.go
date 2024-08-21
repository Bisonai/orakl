package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/reducer"
	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

func NewFetcher(config Config, feeds []Feed, latestFeedDataMap *LatestFeedDataMap, feedDataDumpChannel chan *FeedData) *Fetcher {
	return &Fetcher{
		Config:              config,
		Feeds:               feeds,
		fetcherCtx:          nil,
		cancel:              nil,
		latestFeedDataMap:   latestFeedDataMap,
		FeedDataDumpChannel: feedDataDumpChannel,
	}
}

func (f *Fetcher) Run(ctx context.Context, proxies []Proxy) {
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
				err := f.fetcherJob(f.fetcherCtx, proxies)
				if err != nil && !errors.Is(err, errorSentinel.ErrFetcherNoDataFetched) {
					log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetchAndInsert")
				}
			}
		}
	}()
}

func (f *Fetcher) fetcherJob(ctx context.Context, proxies []Proxy) error {
	log.Debug().Str("Player", "Fetcher").Str("fetcher", f.Name).Msg("fetcherJob")
	result, err := f.fetch(proxies)
	if err != nil && !errors.Is(err, errorSentinel.ErrFetcherNoDataFetched) {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in fetch")
		return err
	}

	if len(result) == 0 {
		return errorSentinel.ErrFetcherNoDataFetched
	}

	err = f.latestFeedDataMap.SetLatestFeedData(result)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in SetLatestFeedData")
		return err
	}

	for _, feedData := range result {
		f.FeedDataDumpChannel <- feedData
	}
	return nil
}

func (f *Fetcher) fetch(proxies []Proxy) ([]*FeedData, error) {
	feeds := f.Feeds

	data := []*FeedData{}
	errList := []error{}
	dataChan := make(chan *FeedData)
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
			default:
				errChan <- errorSentinel.ErrFetcherInvalidType
				return
			}
			now := time.Now()
			dataChan <- &FeedData{FeedID: feed.ID, Value: resultValue, Timestamp: &now, Volume: 0}

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
	return request.Request[interface{}](request.WithEndpoint(*definition.Url), request.WithHeaders(definition.Headers))
}

func (f *Fetcher) requestWithProxy(definition *Definition, proxyUrl string) (interface{}, error) {
	return request.Request[interface{}](request.WithEndpoint(*definition.Url), request.WithHeaders(definition.Headers), request.WithProxy(proxyUrl))
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
