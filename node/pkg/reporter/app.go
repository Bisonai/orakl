package reporter

import (
	"context"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func New() *App {
	return &App{
		Reporters: []*Reporter{},
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.setReporters(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set reporters")
		return err
	}

	return a.startReporters(ctx)
}

func (a *App) setReporters(ctx context.Context) error {
	dalApiKey := os.Getenv("API_KEY")
	if dalApiKey == "" {
		return errorSentinel.ErrReporterDalApiKeyNotFound
	}

	dalWsEndpoint := os.Getenv("DAL_WS_URL")
	if dalWsEndpoint == "" {
		dalWsEndpoint = "ws://orakl-dal.orakl.svc.cluster.local/ws"
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return errorSentinel.ErrReporterSubmissionProxyContractNotFound
	}

	chainHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create chain helper")
		return err
	}
	a.chainHelper = chainHelper

	cachedWhitelist, err := ReadOnchainWhitelist(ctx, chainHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get whitelist, starting with empty whitelist")
		cachedWhitelist = []common.Address{}
	}

	configs, err := getConfigs(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get reporter configs")
		return err
	}

	dalWsHelper, dalWsHelperErr := SetupDalWsHelper(ctx, configs, dalWsEndpoint, dalApiKey)
	if dalWsHelperErr != nil {
		return dalWsHelperErr
	}
	a.WsHelper = dalWsHelper

	groupedConfigs := groupConfigsBySubmitIntervals(configs)
	for groupInterval, configs := range groupedConfigs {
		reporter, errNewReporter := NewReporter(
			ctx,
			WithConfigs(configs),
			WithInterval(groupInterval),
			WithContractAddress(contractAddress),
			WithCachedWhitelist(cachedWhitelist),
		)
		if errNewReporter != nil {
			log.Error().Str("Player", "Reporter").Err(errNewReporter).Msg("failed to set reporter")
			return errNewReporter
		}
		reporter.LatestData = &a.LatestData
		a.Reporters = append(a.Reporters, reporter)
	}
	if len(a.Reporters) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no reporters set")
		return errorSentinel.ErrReporterNotFound
	}

	deviationReporter, errNewDeviationReporter := NewReporter(
		ctx,
		WithConfigs(configs),
		WithInterval(DEVIATION_INTERVAL),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(cachedWhitelist),
		WithJobType(DeviationJob),
	)
	if errNewDeviationReporter != nil {
		log.Error().Str("Player", "Reporter").Err(errNewDeviationReporter).Msg("failed to set deviation reporter")
		return errNewDeviationReporter
	}
	a.Reporters = append(a.Reporters, deviationReporter)

	log.Info().Str("Player", "Reporter").Msgf("%d reporters set", len(a.Reporters))
	return nil
}

func (a *App) startReporters(ctx context.Context) error {
	var errs []string

	go a.WsHelper.Run(ctx, a.handleWsMessage)

	for _, reporter := range a.Reporters {
		err := a.startReporter(ctx, reporter)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("failed to start reporter")
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errorSentinel.ErrReporterStart
	}

	return nil
}

func (a *App) startReporter(ctx context.Context, reporter *Reporter) error {
	if reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter already running")
		return errorSentinel.ErrReporterAlreadyRunning
	}

	reporter.KaiaHelper = a.chainHelper

	nodeCtx, cancel := context.WithCancel(ctx)
	reporter.nodeCtx = nodeCtx
	reporter.nodeCancel = cancel
	reporter.isRunning = true

	go reporter.Run(nodeCtx)
	return nil
}

func getConfigs(ctx context.Context) ([]Config, error) {
	reporterConfigs, err := db.QueryRows[Config](ctx, GET_REPORTER_CONFIGS, nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to load reporter configs")
		return nil, err
	}
	return reporterConfigs, nil
}

func groupConfigsBySubmitIntervals(reporterConfigs []Config) map[int][]Config {
	grouped := make(map[int][]Config)
	for _, sa := range reporterConfigs {
		var interval = 5000
		if sa.SubmitInterval != nil && *sa.SubmitInterval > 0 {
			interval = *sa.SubmitInterval
		}
		grouped[interval] = append(grouped[interval], sa)
	}
	return grouped
}

func (a *App) handleWsMessage(ctx context.Context, data map[string]interface{}) error {
	a.LatestData.Store(data["symbol"], data)
	return nil
}

func (a *App) GetReporterWithInterval(interval int) (*Reporter, error) {
	for _, reporter := range a.Reporters {
		if reporter.SubmissionInterval == time.Duration(interval)*time.Millisecond {
			return reporter, nil
		}
	}
	return nil, errorSentinel.ErrReporterNotFound
}
