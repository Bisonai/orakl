package reporter

import (
	"context"
	"os"
	"sort"
	"sync"

	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/configfetch"
	"bisonai.com/miko/node/pkg/utils/configreload"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func New() *App {
	return &App{
		Reporters:              []*Reporter{},
		LatestDataMap:          new(sync.Map),
		LatestSubmittedDataMap: new(sync.Map),
	}
}

func (a *App) Run(ctx context.Context) error {
	configs, err := a.setReporters(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set reporters")
		return err
	}

	a.startReporters(ctx)

	reloader := configreload.New(configfetch.FeedsPath(chainName()), len(configs),
		func(c Config) string { return c.Name }, a.applyConfigs)
	go reloader.Run(ctx, configreload.Interval(), configreload.NotifyHUP())

	return nil
}

func (a *App) setReporters(ctx context.Context) ([]Config, error) {
	dalApiKey := secrets.GetSecret("API_KEY")
	if dalApiKey == "" {
		return nil, errorSentinel.ErrReporterDalApiKeyNotFound
	}

	dalWsEndpoint := os.Getenv("DAL_WS_URL")
	if dalWsEndpoint == "" {
		dalWsEndpoint = "ws://orakl-dal.orakl.svc.cluster.local/ws"
	}

	dalRestEndpoint := os.Getenv("DAL_REST_URL")
	if dalRestEndpoint == "" {
		return nil, errorSentinel.ErrReporterDalRestEndpointNotFound
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return nil, errorSentinel.ErrReporterSubmissionProxyContractNotFound
	}

	chainHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create chain helper")
		return nil, err
	}

	cachedWhitelist, err := ReadOnchainWhitelist(ctx, chainHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get whitelist, starting with empty whitelist")
		cachedWhitelist = []common.Address{}
	}

	configs, err := fetchConfigs()
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get reporter configs")
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.deps = reporterDeps{
		contractAddress: contractAddress,
		whitelist:       cachedWhitelist,
		chainHelper:     chainHelper,
		dalRestEndpoint: dalRestEndpoint,
		dalWsEndpoint:   dalWsEndpoint,
		dalApiKey:       dalApiKey,
	}

	if err := a.buildReporters(ctx, configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// buildReporters creates the DAL websocket helper, the interval reporters and
// the deviation reporter for configs. Caller holds a.mu.
func (a *App) buildReporters(ctx context.Context, configs []Config) error {
	dalWsHelper, dalWsHelperErr := SetupDalWsHelper(ctx, configs, a.deps.dalWsEndpoint, a.deps.dalApiKey)
	if dalWsHelperErr != nil {
		return dalWsHelperErr
	}
	a.WsHelper = dalWsHelper

	a.groups = map[int]*Reporter{}
	groupedConfigs := groupConfigsBySubmitIntervals(configs)
	for groupInterval, configs := range groupedConfigs {
		reporter, errNewReporter := a.newReporter(ctx, groupInterval, configs, ReportJob)
		if errNewReporter != nil {
			log.Error().Str("Player", "Reporter").Err(errNewReporter).Msg("failed to set reporter")
			return errNewReporter
		}
		a.groups[groupInterval] = reporter
	}
	if len(a.groups) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no reporters set")
		return errorSentinel.ErrReporterNotFound
	}

	deviationReporter, errNewDeviationReporter := a.newReporter(ctx, DEVIATION_INTERVAL, configs, DeviationJob)
	if errNewDeviationReporter != nil {
		log.Error().Str("Player", "Reporter").Err(errNewDeviationReporter).Msg("failed to set deviation reporter")
		return errNewDeviationReporter
	}
	a.deviation = deviationReporter
	a.pairs = pairIntervals(configs)
	a.configs = configs
	a.rebuildReporterList()

	log.Info().Str("Player", "Reporter").Msgf("%d reporters set", len(a.Reporters))
	return nil
}

func (a *App) newReporter(ctx context.Context, interval int, configs []Config, jobType JobType) (*Reporter, error) {
	return NewReporter(
		ctx,
		WithConfigs(configs),
		WithInterval(interval),
		WithContractAddress(a.deps.contractAddress),
		WithCachedWhitelist(a.deps.whitelist),
		WithJobType(jobType),
		WithKaiaHelper(a.deps.chainHelper),
		WithLatestDataMap(a.LatestDataMap),
		WithLatestSubmittedDataMap(a.LatestSubmittedDataMap),
		WithDalRestEndpoint(a.deps.dalRestEndpoint),
	)
}

// rebuildReporterList sets Reporters to the regular reporters (by interval)
// followed by the deviation reporter. Caller holds a.mu.
func (a *App) rebuildReporterList() {
	intervals := make([]int, 0, len(a.groups))
	for interval := range a.groups {
		intervals = append(intervals, interval)
	}
	sort.Ints(intervals)
	a.Reporters = make([]*Reporter, 0, len(intervals)+1)
	for _, interval := range intervals {
		a.Reporters = append(a.Reporters, a.groups[interval])
	}
	if a.deviation != nil {
		a.Reporters = append(a.Reporters, a.deviation)
	}
}

func (a *App) startReporters(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	wsCtx, wsCancel := context.WithCancel(ctx)
	a.wsCancel = wsCancel
	go a.WsHelper.Run(wsCtx, a.HandleWsMessage)

	for _, reporter := range a.Reporters {
		go reporter.Run(ctx)
	}
}

func chainName() string {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Info().Str("Player", "Reporter").Msg("CHAIN env not set, defaulting to kairos")
		chain = "kairos"
	}
	return chain
}

func fetchConfigs() ([]Config, error) {
	return configfetch.Get[[]Config](context.Background(), configfetch.FeedsPath(chainName()))
}

func groupConfigsBySubmitIntervals(reporterConfigs []Config) map[int][]Config {
	grouped := make(map[int][]Config)
	for _, sa := range reporterConfigs {
		interval := submitInterval(sa)
		grouped[interval] = append(grouped[interval], sa)
	}
	return grouped
}

func (a *App) HandleWsMessage(ctx context.Context, data map[string]interface{}) error {
	submissionData, err := ProcessDalWsRawData(data)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to process dal ws raw data")
		return err
	}
	a.LatestDataMap.Store(data["symbol"], submissionData)
	return nil
}
