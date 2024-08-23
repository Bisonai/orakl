package reporter

import (
	"context"
	"fmt"
	"os"
	"sync"

	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/klaytn/klaytn/common"
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
	err := a.setReporters(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set reporters")
		return err
	}

	a.startReporters(ctx)

	return nil
}

func (a *App) setReporters(ctx context.Context) error {
	dalApiKey := secrets.GetSecret("API_KEY")
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

	chainHelper, err := helper.NewChainHelper(ctx, helper.WithoutAdditionalProviderUrls())
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create chain helper")
		return err
	}

	cachedWhitelist, err := ReadOnchainWhitelist(ctx, chainHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get whitelist, starting with empty whitelist")
		cachedWhitelist = []common.Address{}
	}

	configs, err := fetchConfigs()
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
			WithKaiaHelper(chainHelper),
			WithLatestDataMap(a.LatestDataMap),
			WithLatestSubmittedDataMap(a.LatestSubmittedDataMap),
		)
		if errNewReporter != nil {
			log.Error().Str("Player", "Reporter").Err(errNewReporter).Msg("failed to set reporter")
			return errNewReporter
		}
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
		WithKaiaHelper(chainHelper),
		WithLatestDataMap(a.LatestDataMap),
		WithLatestSubmittedDataMap(a.LatestSubmittedDataMap),
	)
	if errNewDeviationReporter != nil {
		log.Error().Str("Player", "Reporter").Err(errNewDeviationReporter).Msg("failed to set deviation reporter")
		return errNewDeviationReporter
	}
	a.Reporters = append(a.Reporters, deviationReporter)

	log.Info().Str("Player", "Reporter").Msgf("%d reporters set", len(a.Reporters))
	return nil
}

func (a *App) startReporters(ctx context.Context) {
	go a.WsHelper.Run(ctx, a.HandleWsMessage)

	for _, reporter := range a.Reporters {
		go reporter.Run(ctx)
	}
}

func fetchConfigs() ([]Config, error) {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Info().Str("Player", "Reporter").Msg("CHAIN env not set, defaulting to baobab")
		chain = "baobab"
	}
	endpoint := fmt.Sprintf("https://config.orakl.network/%s_configs.json", chain)
	configs, err := request.Request[[]Config](request.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return configs, nil
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

func (a *App) HandleWsMessage(ctx context.Context, data map[string]interface{}) error {
	submissionData, err := ProcessDalWsRawData(data)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to process dal ws raw data")
		return err
	}
	a.LatestDataMap.Store(data["symbol"], submissionData)
	return nil
}
