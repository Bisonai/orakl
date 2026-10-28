package fetcher

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/metrics"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"bisonai.com/miko/node/pkg/websocketfetcher"
	"github.com/rs/zerolog/log"
)

// New builds the fetcher app on the process-wide config store.
func New(bus *bus.MessageBus) *App {
	return &App{
		Configs:          nodeconfig.Default(),
		Fetchers:         make(map[int32]*Fetcher, 0),
		WebsocketFetcher: websocketfetcher.New(),
		LatestFeedDataMap: &LatestFeedDataMap{
			FeedDataMap: make(map[int32]*FeedData),
			Mu:          sync.RWMutex{},
		},
		LocalAggregateValueMap: NewLocalAggregateValueMap(),
		Bus:                    bus,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()

	err := a.initialize(ctx)
	if err != nil {
		return err
	}

	a.subscribe(ctx)
	// once per app lifetime; initialize re-runs on every refresh
	go a.LatestFeedDataMap.CleanupJob(ctx)

	return a.startAll(ctx)
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
	case bus.STOP_FETCHER_APP, bus.START_FETCHER_APP, bus.REFRESH_FETCHER_APP:
		if !a.lifecycleMu.TryLock() {
			bus.HandleMessageError(errorSentinel.ErrFetcherLifecycleInProgress, msg, "rejected fetcher command: "+msg.Content.Command)
			return
		}
		// handlers below reply into a buffered chan; forward it only after
		// unlocking so a slow/absent reader can't pin lifecycleMu
		reply := msg.Response
		buffered := make(chan bus.MessageResponse, 1)
		msg.Response = buffered
		defer func() {
			a.lifecycleMu.Unlock()
			select {
			case resp := <-buffered:
				reply <- resp
			default:
			}
		}()
	}

	switch msg.Content.Command {
	case bus.STOP_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("stopping all fetchers")
		err := a.stopAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.START_FETCHER_APP:
		log.Debug().Str("Player", "Fetcher").Msg("starting all fetchers")
		err := a.startAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_FETCHER_APP:
		// build the new set before stopping the running one, so a failed build
		// (e.g. bad proxy env, websocket init error) leaves the previous set
		// assigned and running
		state, err := buildFetcherState(a, ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to initialize fetchers")
			bus.HandleMessageError(err, msg, "failed to initialize fetchers")
			return
		}
		err = a.stopAll(ctx)
		if err != nil {
			state.websocketFetcher.Close()
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to stop all fetchers")
			bus.HandleMessageError(err, msg, "failed to stop all fetchers")
			return
		}
		a.applyState(state)
		err = a.startAll(ctx)
		if err != nil {
			log.Error().Err(err).Str("Player", "Fetcher").Msg("failed to start all fetchers")
			bus.HandleMessageError(err, msg, "failed to start all fetchers")
			return
		}

		log.Debug().Str("Player", "Fetcher").Msg("refreshing fetcher")
		msg.Response <- bus.MessageResponse{Success: true}
	}
}

func (a *App) startAll(ctx context.Context) error {
	err := a.startAllFetchers(ctx)
	if err != nil {
		return err
	}

	err = a.startAllLocalAggregators(ctx)
	if err != nil {
		return err
	}

	a.WebsocketFetcher.Start(ctx)
	return nil
}

func (a *App) stopAll(ctx context.Context) error {
	a.WebsocketFetcher.Stop()

	err := a.stopAllFetchers(ctx)
	if err != nil {
		return err
	}

	return a.stopAllLocalAggregators(ctx)
}

func (a *App) startFetcher(ctx context.Context, fetcher *Fetcher) error {
	if fetcher.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher already running")
		return nil
	}

	fetcher.Run(ctx, a.Proxies)

	log.Debug().Str("Player", "Fetcher").Str("fetcher", fetcher.Name).Msg("fetcher started")
	return nil
}

func (a *App) startLocalAggregator(ctx context.Context, localAggregator *LocalAggregator) error {
	if localAggregator.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator already running")
		return nil
	}

	localAggregator.Run(ctx)

	log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator started")
	return nil
}

func (a *App) startAllFetchers(ctx context.Context) error {
	for _, fetcher := range a.Fetchers {
		err := a.startFetcher(ctx, fetcher)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("fetcher", fetcher.Name).Msg("failed to start fetcher")
			return err
		}
		// starts with random sleep to avoid all fetchers starting at the same time
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+50))
	}
	return nil
}

func (a *App) startAllLocalAggregators(ctx context.Context) error {
	for _, localAggregator := range a.LocalAggregators {
		err := a.startLocalAggregator(ctx, localAggregator)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("localAggregator", localAggregator.Name).Msg("failed to start localAggregator")
			return err
		}
		// starts with random sleep to avoid all fetchers starting at the same time
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+50))
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

func (a *App) stopLocalAggregator(ctx context.Context, localAggregator *LocalAggregator) error {
	log.Debug().Str("LocalAggregator", localAggregator.Name).Msg("stopping LocalAggregator")
	if !localAggregator.isRunning {
		log.Debug().Str("Player", "Fetcher").Str("localAggregator", localAggregator.Name).Msg("localAggregator already stopped")
		return nil
	}
	if localAggregator.cancel == nil {
		return errorSentinel.ErrLocalAggregatorCancelNotFound
	}
	localAggregator.cancel()
	localAggregator.isRunning = false
	return nil
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

func (a *App) stopAllLocalAggregators(ctx context.Context) error {
	for _, localAggregator := range a.LocalAggregators {
		err := a.stopLocalAggregator(ctx, localAggregator)
		if err != nil {
			log.Error().Str("Player", "Fetcher").Err(err).Str("localAggregator", localAggregator.Name).Msg("failed to stop localAggregator")
			return err
		}
	}
	return nil
}

// fetcherState is a fully built fetcher set, swapped into the App only once
// every part of it was built.
type fetcherState struct {
	fetchers         map[int32]*Fetcher
	localAggregators map[int32]*LocalAggregator
	proxies          []Proxy
	websocketFetcher *websocketfetcher.App
}

// buildFetcherState is a seam so tests can inject build failures.
var buildFetcherState = (*App).buildState

func (a *App) initialize(ctx context.Context) error {
	state, err := buildFetcherState(a, ctx)
	if err != nil {
		return err
	}
	a.applyState(state)
	return nil
}

// applyState swaps in a built set; the previous set must already be stopped.
func (a *App) applyState(state *fetcherState) {
	// release the replaced websocket fetcher's chain reader
	if a.WebsocketFetcher != nil {
		a.WebsocketFetcher.Close()
	}

	a.Fetchers = state.fetchers
	a.LocalAggregators = state.localAggregators
	a.Proxies = state.proxies
	a.WebsocketFetcher = state.websocketFetcher

	names := make([]string, 0, len(state.localAggregators))
	for _, localAggregator := range state.localAggregators {
		names = append(names, localAggregator.Name)
	}
	metrics.SetLocalAggregateConfigs(names)
}

func (a *App) buildState(ctx context.Context) (*fetcherState, error) {
	proxies, err := loadProxies()
	if err != nil {
		return nil, err
	}

	configs := a.Configs.ConfigsWithFeeds()
	fetchers := make(map[int32]*Fetcher, len(configs))
	localAggregators := make(map[int32]*LocalAggregator, len(configs))

	for _, config := range configs {
		// fetcher gets only plain HTTP feeds (no websocket/DEX feeds)
		if len(config.HTTPFeeds) > 0 {
			fetchers[config.ID] = NewFetcher(config.Config, config.HTTPFeeds, a.LatestFeedDataMap)
		}

		// for localAggregator it'll get all feeds to be collected
		localAggregators[config.ID] = NewLocalAggregator(config.Config, config.Feeds, a.Bus, a.LatestFeedDataMap, a.LocalAggregateValueMap)
	}

	// fresh instance so Init doesn't touch the running one
	websocketFetcher := websocketfetcher.New()
	err = websocketFetcher.Init(ctx, websocketfetcher.WithLatestFeedDataMap(a.LatestFeedDataMap), websocketfetcher.WithConfigStore(a.Configs))
	if err != nil {
		websocketFetcher.Close()
		return nil, err
	}

	return &fetcherState{
		fetchers:         fetchers,
		localAggregators: localAggregators,
		proxies:          proxies,
		websocketFetcher: websocketFetcher,
	}, nil
}

// loadProxies parses EnvProxies; unset or empty means no proxies.
func loadProxies() ([]Proxy, error) {
	raw := strings.TrimSpace(os.Getenv(EnvProxies))
	if raw == "" {
		return nil, nil
	}
	var proxies []Proxy
	for i, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		var location *string
		if loc, rest, ok := strings.Cut(entry, "="); ok {
			loc = strings.TrimSpace(loc)
			location = &loc
			entry = strings.TrimSpace(rest)
		}
		u, err := url.Parse(entry)
		if err != nil || u.Scheme == "" || u.Hostname() == "" || u.Path != "" || u.RawQuery != "" || u.User != nil {
			return nil, fmt.Errorf("%s: invalid proxy %q, want [location=]protocol://host:port", EnvProxies, entry)
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("%s: invalid port in proxy %q", EnvProxies, entry)
		}
		proxies = append(proxies, Proxy{ID: int64(i + 1), Protocol: u.Scheme, Host: u.Hostname(), Port: port, Location: location})
	}
	return proxies, nil
}

func (a *App) String() string {
	return fmt.Sprintf("%+v\n", a.Fetchers)
}
