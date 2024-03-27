package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	chain_helper "bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	"bisonai.com/orakl/node/pkg/utils/reducer"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

const FETCHER_FREQUENCY = 2 * time.Second

func New(bus *bus.MessageBus) *App {
	return &App{
		Fetchers: make(map[int64]*Fetcher, 0),
		Bus:      bus,
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.initialize(ctx)
	if err != nil {
		return err
	}

	a.subscribe(ctx)

	err = a.startAllFetchers(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Fetcher").Msg("fetcher subscribing to message bus")
	channel := a.Bus.Subscribe(bus.FETCHER)
	go func() {
		log.Debug().Str("Player", "Fetcher").Msg("fetcher message bus subscription goroutine started")
		for {
			select {
			case msg := <-channel:
				log.Debug().
					Str("Player", "Fetcher").
					Str("from", msg.From).
					Str("to", msg.To).
					Str("command", msg.Content.Command).
					Msg("fetcher received message")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Fetcher").Msg("fetcher message bus subscription goroutine stopped")
				return
			}
		}
	}()
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	if msg.From != bus.ADMIN {
		log.Debug().Str("Player", "Fetcher").Msg("fetcher received message from non-admin")
		return
	}

	if msg.To != bus.FETCHER {
		log.Debug().Str("Player", "Fetcher").Msg("message not for fetcher")
		return
	}

	switch msg.Content.Command {
	case bus.ACTIVATE_FETCHER:
		log.Debug().Str("Player", "Fetcher").Msg("activate fetcher msg received")
		adapterId, err := bus.ParseInt64MsgParam(msg, "id")
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to parse adapterId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int64("adapterId", adapterId).Msg("activating fetcher")
		err = a.startFetcherById(ctx, adapterId)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_FETCHER:
		log.Debug().Str("Player", "Fetcher").Msg("deactivate fetcher msg received")
		adapterId, err := bus.ParseInt64MsgParam(msg, "id")
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to parse adapterId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int64("adapterId", adapterId).Msg("deactivating fetcher")
		err = a.stopFetcherById(ctx, adapterId)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.STOP_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("stopping all fetchers")
		err := a.stopAllFetchers(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.START_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("starting all fetchers")
		err := a.startAllFetchers(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_FETCHER_APP:
		err := a.stopAllFetchers(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		err = a.initialize(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to initialize fetchers")
			return
		}
		err = a.startAllFetchers(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}

		log.Debug().Str("Player", "Fetcher").Msg("refreshing fetcher")
		msg.Response <- bus.MessageResponse{Success: true}
	}
}

func (a *App) startFetcher(ctx context.Context, fetcher *Fetcher) error {
	if fetcher.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher already running")
		return nil
	}
	fetcherCtx, cancel := context.WithCancel(ctx)
	fetcher.fetcherCtx = fetcherCtx
	fetcher.cancel = cancel
	fetcher.isRunning = true

	ticker := time.NewTicker(FETCHER_FREQUENCY)

	go func() {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("starting fetcher goroutine")
		for {
			select {
			case <-ticker.C:
				log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetching and inserting")
				err := a.fetchAndInsert(fetcherCtx, *fetcher)
				if err != nil {
					log.Error().Str("Player", "Fetcher").Err(err).Msg("failed to fetch and insert")
				}
			case <-fetcherCtx.Done():
				log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher stopped")
				return
			}
		}
	}()

	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher started")
	return nil
}

func (a *App) startFetcherById(ctx context.Context, adapterId int64) error {
	if fetcher, ok := a.Fetchers[adapterId]; ok {
		return a.startFetcher(ctx, fetcher)
	}
	return errors.New("fetcher not found by id:" + strconv.FormatInt(adapterId, 10))
}

func (a *App) startAllFetchers(ctx context.Context) error {
	for _, fetcher := range a.Fetchers {
		err := a.startFetcher(ctx, fetcher)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("fetcher", fetcher.Name).Msg("failed to start fetcher")
			return err
		}
		// starts with random sleep to avoid all fetchers starting at the same time
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(300)+100))
	}
	return nil
}

func (a *App) stopFetcher(ctx context.Context, fetcher *Fetcher) error {
	log.Debug().Str("fetcher", fetcher.Name).Msg("stopping fetcher")
	if !fetcher.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher already stopped")
		return nil
	}
	if fetcher.cancel == nil {
		return errors.New("fetcher cancel function not found")
	}
	fetcher.cancel()
	fetcher.isRunning = false
	return nil
}

func (a *App) stopFetcherById(ctx context.Context, adapterId int64) error {
	if fetcher, ok := a.Fetchers[adapterId]; ok {
		return a.stopFetcher(ctx, fetcher)
	}
	return errors.New("fetcher not found by id:" + strconv.FormatInt(adapterId, 10))
}

func (a *App) stopAllFetchers(ctx context.Context) error {
	for _, fetcher := range a.Fetchers {
		err := a.stopFetcher(ctx, fetcher)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("fetcher", fetcher.Name).Msg("failed to stop fetcher")
			return err
		}
	}
	return nil
}

func (a *App) fetchAndInsert(ctx context.Context, fetcher Fetcher) error {
	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetching and inserting")

	results, err := a.fetch(fetcher)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetch complete")

	aggregated, err := calculator.GetFloatMed(results)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Float64("aggregated", aggregated).Msg("aggregated")

	err = a.insertPgsql(ctx, fetcher.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("inserted into pgsql")

	err = a.insertRdb(ctx, fetcher.Name, aggregated)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("inserted into rdb")
	return nil
}

func (a *App) insertPgsql(ctx context.Context, name string, value float64) error {
	err := db.QueryWithoutResult(ctx, InsertLocalAggregateQuery, map[string]any{"name": name, "value": int64(value)})
	return err
}

func (a *App) insertRdb(ctx context.Context, name string, value float64) error {
	key := "localAggregate:" + name
	data, err := json.Marshal(redisAggregate{Value: int64(value), Timestamp: time.Now()})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func (a *App) fetch(fetcher Fetcher) ([]float64, error) {
	feeds := fetcher.Feeds

	data := []float64{}
	dataChan := make(chan float64)
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

			var result float64
			var fetchErr error

			switch {
			case definition.Type == nil:
				result, fetchErr = a.cex(*definition)
				if fetchErr != nil {
					errChan <- fetchErr
					return
				}
			case *definition.Type == "UniswapPool":
				result, fetchErr = a.uniswapV3(*definition)
				if fetchErr != nil {
					errChan <- fetchErr
					return
				}
			default:
				errChan <- errors.New("unknown fetcher type")
			}

			dataChan <- result
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

func (a *App) cex(definition Definition) (float64, error) {
	rawResult, err := a.requestFeed(definition)
	if err != nil {
		log.Error().Str("Player", "Fetcher").Err(err).Msg("error in requestFeed")
		return 0, err
	}

	return reducer.Reduce(rawResult, definition.Reducers)
}

func (a *App) uniswapV3(definition Definition) (float64, error) {
	if definition.Address == nil || definition.ChainID == nil || definition.Token0Decimals == nil || definition.Token1Decimals == nil {
		log.Error().Any("definition", definition).Msg("missing required fields for uniswapV3")
		return 0, errors.New("missing required fields for uniswapV3")
	}

	helper := a.ChainHelpers[*definition.ChainID]
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

func (a *App) requestFeed(definition Definition) (interface{}, error) {
	var proxies []Proxy
	if definition.Location != nil && *definition.Location != "" {
		proxies = a.filterProxyByLocation(*definition.Location)
	} else {
		proxies = a.Proxies
	}

	if len(proxies) > 0 {
		proxy := proxies[rand.Intn(len(proxies))]
		proxyUrl := fmt.Sprintf("%s://%s:%d", proxy.Protocol, proxy.Host, proxy.Port)
		log.Debug().Str("Player", "Fetcher").Str("proxyUrl", proxyUrl).Msg("using proxy")
		return a.requestWithProxy(definition, proxyUrl)
	}

	return a.requestWithoutProxy(definition)
}

func (a *App) requestWithoutProxy(definition Definition) (interface{}, error) {
	return request.GetRequest[interface{}](*definition.Url, nil, definition.Headers)
}

func (a *App) requestWithProxy(definition Definition, proxyUrl string) (interface{}, error) {
	return request.GetRequestProxy[interface{}](*definition.Url, nil, definition.Headers, proxyUrl)
}

func (a *App) filterProxyByLocation(location string) []Proxy {
	proxies := []Proxy{}
	for _, proxy := range a.Proxies {
		if proxy.Location != nil && *proxy.Location == location {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}

func (a *App) getAdapters(ctx context.Context) ([]Adapter, error) {
	adapters, err := db.QueryRows[Adapter](ctx, SelectActiveAdaptersQuery, nil)
	if err != nil {
		return nil, err
	}
	return adapters, err
}

func (a *App) getFeeds(ctx context.Context, adapterId int64) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByAdapterIdQuery, map[string]any{"adapterId": adapterId})
	if err != nil {
		return nil, err
	}

	return feeds, err
}

func (a *App) getProxies(ctx context.Context) ([]Proxy, error) {
	proxies, err := db.QueryRows[Proxy](ctx, SelectAllProxiesQuery, nil)
	if err != nil {
		return nil, err
	}
	return proxies, err
}

func (a *App) initialize(ctx context.Context) error {
	adapters, err := a.getAdapters(ctx)
	if err != nil {
		return err
	}
	a.Fetchers = make(map[int64]*Fetcher, len(adapters))
	for _, adapter := range adapters {
		feeds, err := a.getFeeds(ctx, adapter.ID)
		if err != nil {
			return err
		}

		a.Fetchers[adapter.ID] = &Fetcher{
			Adapter:   adapter,
			Feeds:     feeds,
			isRunning: false,
		}
	}

	proxies, getProxyErr := a.getProxies(ctx)
	if getProxyErr != nil {
		return getProxyErr
	}
	a.Proxies = proxies

	if a.ChainHelpers != nil && len(a.ChainHelpers) > 0 {
		for _, chainHelper := range a.ChainHelpers {
			chainHelper.Close()
		}
	}

	chainHelpers, getChainHelpersErr := a.getChainHelpers(ctx)
	if getChainHelpersErr != nil {
		return getChainHelpersErr
	}
	a.ChainHelpers = chainHelpers

	return nil
}

func (a *App) getChainHelpers(ctx context.Context) (map[string]ChainHelper, error) {
	cypressProviderUrl := os.Getenv("FETCHER_CYPRESS_PROVIDER_URL")
	if cypressProviderUrl == "" {
		log.Info().Msg("cypress provider url not set, using default url: https://public-en-cypress.klaytn.net")
		cypressProviderUrl = "https://public-en-cypress.klaytn.net"
	}

	cypressHelper, err := chain_helper.NewKlayHelper(ctx, cypressProviderUrl)
	if err != nil {
		log.Error().Err(err).Msg("failed to create cypress helper")
		return nil, err
	}

	ethereumProviderUrl := os.Getenv("FETCHER_ETHEREUM_PROVIDER_URL")
	if ethereumProviderUrl == "" {
		log.Info().Msg("ethereum provider url not set, using default url: https://ethereum-mainnet-rpc.allthatnode.com")
		ethereumProviderUrl = "https://ethereum-mainnet-rpc.allthatnode.com"
	}

	ethereumHelper, err := chain_helper.NewEthHelper(ctx, ethereumProviderUrl)
	if err != nil {
		log.Error().Err(err).Msg("failed to create ethereum helper")
		return nil, err
	}

	var result = make(map[string]ChainHelper, 2)

	cypressChainId := cypressHelper.ChainID()
	result[cypressChainId.String()] = cypressHelper

	ethereumChainId := ethereumHelper.ChainID()
	result[ethereumChainId.String()] = ethereumHelper

	return result, nil
}

func (a *App) String() string {
	return fmt.Sprintf("%+v\n", a.Fetchers)
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
