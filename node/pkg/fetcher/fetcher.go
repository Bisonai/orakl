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
	"bisonai.com/miko/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

const (
	FetcherRequestTimeout  = 5 * time.Second
	FetcherMaxRetries      = 3
	FetcherRetryInitDelay  = 500 * time.Millisecond
	FetcherRetryMaxDelay   = 2 * time.Second
)

func NewFetcher(config Config, feeds []Feed, latestFeedDataMap *LatestFeedDataMap, feedDataDumpChannel chan *FeedData) *Fetcher {
	return &Fetcher{
		Config:              config,
		Feeds:               feeds,
		fetcherCtx:          nil,
		cancel:              nil,
		latestFeedDataMap:   latestFeedDataMap,
		FeedDataDumpChannel: feedDataDumpChannel,
		circuitBreakers:     newCircuitBreakerMap(),
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

	activeFeeds := make([]Feed, 0, len(feeds))
	for _, feed := range feeds {
		if f.circuitBreakers.isOpen(feed.ID) {
			log.Debug().Str("Player", "Fetcher").Str("feed", feed.Name).Msg("circuit breaker open, skipping feed")
			continue
		}
		activeFeeds = append(activeFeeds, feed)
	}

	if len(activeFeeds) == 0 {
		return nil, errorSentinel.ErrFetcherNoDataFetched
	}

	data := []*FeedData{}
	errList := []error{}

	type fetchResult struct {
		feedID   int32
		feedName string
		data     *FeedData
		err      error
	}
	resultChan := make(chan fetchResult, len(activeFeeds))

	for _, feed := range activeFeeds {
		go func(feed Feed) {
			definition := new(Definition)
			err := json.Unmarshal(feed.Definition, &definition)
			if err != nil {
				resultChan <- fetchResult{feedID: feed.ID, feedName: feed.Name, err: err}
				return
			}

			var resultValue float64
			var fetchErr error

			switch {
			case definition.Type == nil:
				resultValue, fetchErr = f.cex(definition, proxies)
				if fetchErr != nil {
					resultChan <- fetchResult{feedID: feed.ID, feedName: feed.Name, err: fetchErr}
					return
				}
			default:
				resultChan <- fetchResult{feedID: feed.ID, feedName: feed.Name, err: errorSentinel.ErrFetcherInvalidType}
				return
			}
			now := time.Now()
			resultChan <- fetchResult{feedID: feed.ID, feedName: feed.Name, data: &FeedData{FeedID: feed.ID, Value: resultValue, Timestamp: &now, Volume: 0}}
		}(feed)
	}

	for i := 0; i < len(activeFeeds); i++ {
		r := <-resultChan
		if r.err != nil {
			errList = append(errList, r.err)
			if opened := f.circuitBreakers.recordFailure(r.feedID, r.feedName); opened {
				log.Warn().Str("Player", "Fetcher").Str("feed", r.feedName).
					Int("threshold", CircuitBreakerThreshold).
					Dur("cooldown", CircuitBreakerCooldown).
					Msg("circuit breaker opened, feed will be skipped temporarily")
			}
		} else {
			data = append(data, r.data)
			f.circuitBreakers.recordSuccess(r.feedID)
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
	var result interface{}
	err := retrier.Retry(func() error {
		var reqErr error
		result, reqErr = request.Request[interface{}](
			request.WithEndpoint(*definition.Url),
			request.WithHeaders(definition.Headers),
			request.WithTimeout(FetcherRequestTimeout),
		)
		return reqErr
	}, FetcherMaxRetries, FetcherRetryInitDelay, FetcherRetryMaxDelay)
	return result, err
}

func (f *Fetcher) requestWithProxy(definition *Definition, proxyUrl string) (interface{}, error) {
	var result interface{}
	err := retrier.Retry(func() error {
		var reqErr error
		result, reqErr = request.Request[interface{}](
			request.WithEndpoint(*definition.Url),
			request.WithHeaders(definition.Headers),
			request.WithProxy(proxyUrl),
			request.WithTimeout(FetcherRequestTimeout),
		)
		return reqErr
	}, FetcherMaxRetries, FetcherRetryInitDelay, FetcherRetryMaxDelay)
	return result, err
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
