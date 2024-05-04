package fetcher

import (
	"context"
	"fmt"

	"math/rand"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	chain_helper "bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus) *App {
	return &App{
		Fetchers: make(map[int32]*Fetcher, 0),
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
		configId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to parse configId")
			bus.HandleMessageError(err, msg, "failed to parse configId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int32("configId", configId).Msg("activating fetcher")
		err = a.startFetcherById(ctx, configId)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start fetcher")
			bus.HandleMessageError(err, msg, "failed to start fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_FETCHER:
		log.Debug().Str("Player", "Fetcher").Msg("deactivate fetcher msg received")
		configId, err := bus.ParseInt32MsgParam(msg, "id")
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to parse configId")
			bus.HandleMessageError(err, msg, "failed to parse configId")
			return
		}

		log.Debug().Str("Player", "Fetcher").Int32("configId", configId).Msg("deactivating fetcher")
		err = a.stopFetcherById(ctx, configId)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop fetcher")
			bus.HandleMessageError(err, msg, "failed to stop fetcher")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.STOP_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("stopping all fetchers")
		err := a.stopAllFetchers(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.START_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("starting all fetchers")
		err := a.startAllFetchers(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_FETCHER_APP:
		err := a.stopAllFetchers(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		err = a.initialize(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to initialize fetchers")
			bus.HandleMessageError(err, msg, "failed to initialize fetchers")
			return
		}
		err = a.startAllFetchers(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
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

	fetcher.Run(ctx, a.ChainHelpers, a.Proxies)

	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher started")
	return nil
}

func (a *App) startFetcherById(ctx context.Context, configId int32) error {
	if fetcher, ok := a.Fetchers[configId]; ok {
		return a.startFetcher(ctx, fetcher)
	}
	log.Error().Str("Player", "Fetcher").Int32("adapterId", configId).Msg("fetcher not found")
	return errorSentinel.ErrFetcherNotFound
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
		return errorSentinel.ErrFetcherCancelNotFound
	}
	fetcher.cancel()
	fetcher.isRunning = false
	return nil
}

func (a *App) stopFetcherById(ctx context.Context, configId int32) error {
	if fetcher, ok := a.Fetchers[configId]; ok {
		return a.stopFetcher(ctx, fetcher)
	}
	return errorSentinel.ErrFetcherNotFound
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

func (a *App) getFetcherConfigs(ctx context.Context) ([]FetcherConfig, error) {
	configs, err := db.QueryRows[FetcherConfig](ctx, SelectConfigsQuery, nil)
	if err != nil {
		return nil, err
	}
	return configs, err
}

func (a *App) getFeeds(ctx context.Context, configId int32) ([]Feed, error) {
	feeds, err := db.QueryRows[Feed](ctx, SelectFeedsByConfigIdQuery, map[string]any{"config_id": configId})
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
	fetcherConfigs, err := a.getFetcherConfigs(ctx)
	if err != nil {
		return err
	}
	a.Fetchers = make(map[int32]*Fetcher, len(fetcherConfigs))
	for _, config := range fetcherConfigs {
		feeds, err := a.getFeeds(ctx, config.ID)
		if err != nil {
			return err
		}

		a.Fetchers[config.ID] = NewFetcher(config, feeds)
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

	cypressHelper, err := chain_helper.NewChainHelper(ctx, chain_helper.WithProviderUrl(cypressProviderUrl))
	if err != nil {
		log.Error().Err(err).Msg("failed to create cypress helper")
		return nil, err
	}

	ethereumProviderUrl := os.Getenv("FETCHER_ETHEREUM_PROVIDER_URL")
	if ethereumProviderUrl == "" {
		log.Info().Msg("ethereum provider url not set, using default url: https://ethereum-mainnet-rpc.allthatnode.com")
		ethereumProviderUrl = "https://ethereum-mainnet-rpc.allthatnode.com"
	}

	ethereumHelper, err := chain_helper.NewChainHelper(ctx, chain_helper.WithBlockchainType(chain_helper.Ethereum), chain_helper.WithProviderUrl(ethereumProviderUrl))
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
