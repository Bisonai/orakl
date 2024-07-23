package reporter

import (
	"context"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn/common"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func New(bus *bus.MessageBus, h host.Host, ps *pubsub.PubSub) *App {
	return &App{
		Reporters: []*Reporter{},
		Bus:       bus,
		Host:      h,
		Pubsub:    ps,
	}
}

func (a *App) Run(ctx context.Context) error {
	err := a.setReporters(ctx, a.Host, a.Pubsub)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set reporters")
		return err
	}
	a.subscribe(ctx)

	return a.startReporters(ctx)
}

func (a *App) setReporters(ctx context.Context, h host.Host, ps *pubsub.PubSub) error {
	err := a.clearReporters()
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to clear reporters")
		return err
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return errorSentinel.ErrReporterSubmissionProxyContractNotFound
	}

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create chain helper")
		return err
	}

	cachedWhitelist, err := ReadOnchainWhitelist(ctx, kaiaHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get whitelist, starting with empty whitelist")
		cachedWhitelist = []common.Address{}
	}

	configs, err := getConfigs(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get reporter configs")
		return err
	}

	groupedConfigs := groupConfigsBySubmitIntervals(configs)
	for groupInterval, configs := range groupedConfigs {
		reporter, errNewReporter := NewReporter(
			ctx,
			WithHost(h),
			WithPubsub(ps),
			WithConfigs(configs),
			WithInterval(groupInterval),
			WithContractAddress(contractAddress),
			WithCachedWhitelist(cachedWhitelist),
			WithKaiaHelper(kaiaHelper),
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
		WithHost(h),
		WithPubsub(ps),
		WithConfigs(configs),
		WithInterval(DEVIATION_INTERVAL),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(cachedWhitelist),
		WithJobType(DeviationJob),
		WithKaiaHelper(kaiaHelper),
	)
	if errNewDeviationReporter != nil {
		log.Error().Str("Player", "Reporter").Err(errNewDeviationReporter).Msg("failed to set deviation reporter")
		return errNewDeviationReporter
	}
	a.Reporters = append(a.Reporters, deviationReporter)

	log.Info().Str("Player", "Reporter").Msgf("%d reporters set", len(a.Reporters))
	return nil
}

func (a *App) clearReporters() error {
	if a.Reporters == nil {
		return nil
	}

	var errs []string
	for _, reporter := range a.Reporters {
		if reporter.isRunning {
			err := stopReporter(reporter)
			if err != nil {
				log.Error().Str("Player", "Reporter").Err(err).Msg("failed to stop reporter")
				errs = append(errs, err.Error())
			}
		}
	}
	a.Reporters = make([]*Reporter, 0)

	if len(errs) > 0 {
		return errorSentinel.ErrReporterClear
	}

	return nil
}

func (a *App) startReporters(ctx context.Context) error {
	var errs []string

	for _, reporter := range a.Reporters {
		err := startReporter(ctx, reporter)
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

func (a *App) stopReporters() error {
	var errs []string

	for _, reporter := range a.Reporters {
		err := stopReporter(reporter)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("failed to stop reporter")
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errorSentinel.ErrReporterStop
	}

	return nil
}

func (a *App) subscribe(ctx context.Context) {
	log.Debug().Str("Player", "Reporter").Msg("subscribing to reporter topic")
	channel := a.Bus.Subscribe(bus.REPORTER)
	if channel == nil {
		log.Error().Str("Player", "Reporter").Msg("failed to subscribe to reporter topic")
		return
	}

	go func() {
		log.Debug().Str("Player", "Reporter").Msg("start reporter subscription goroutine")
		for {
			select {
			case msg := <-channel:
				log.Debug().Str("Player", "Reporter").Str("command", msg.Content.Command).Msg("received message from reporter topic")
				go a.handleMessage(ctx, msg)
			case <-ctx.Done():
				log.Debug().Str("Player", "Reporter").Msg("stopping reporter subscription goroutine")
				return
			}
		}
	}()
}

func (a *App) handleMessage(ctx context.Context, msg bus.Message) {
	switch msg.Content.Command {
	case bus.ACTIVATE_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errorSentinel.ErrBusNonAdmin, msg, "reporter received message from non-admin")
			return
		}
		err := a.startReporters(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.DEACTIVATE_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errorSentinel.ErrBusNonAdmin, msg, "reporter received message from non-admin")
			return
		}
		err := a.stopReporters()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	case bus.REFRESH_REPORTER:
		if msg.From != bus.ADMIN {
			bus.HandleMessageError(errorSentinel.ErrBusNonAdmin, msg, "reporter received message from non-admin")
			return
		}
		err := a.stopReporters()
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to stop reporter")
			return
		}

		err = a.setReporters(ctx, a.Host, a.Pubsub)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to set reporters")
			return
		}

		err = a.startReporters(ctx)
		if err != nil {
			bus.HandleMessageError(err, msg, "failed to start reporter")
			return
		}
		msg.Response <- bus.MessageResponse{Success: true}
	}
}

func (a *App) GetReporterWithInterval(interval int) (*Reporter, error) {
	for _, reporter := range a.Reporters {
		if reporter.SubmissionInterval == time.Duration(interval)*time.Millisecond {
			return reporter, nil
		}
	}
	return nil, errorSentinel.ErrReporterNotFound
}

func startReporter(ctx context.Context, reporter *Reporter) error {
	if reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter already running")
		return errorSentinel.ErrReporterAlreadyRunning
	}

	nodeCtx, cancel := context.WithCancel(ctx)
	reporter.nodeCtx = nodeCtx
	reporter.nodeCancel = cancel
	reporter.isRunning = true

	go reporter.Run(nodeCtx)
	return nil
}

func stopReporter(reporter *Reporter) error {
	if !reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter not running")
		return nil
	}

	if reporter.nodeCancel == nil {
		log.Error().Str("Player", "Reporter").Msg("reporter cancel function not found")
		return errorSentinel.ErrReporterCancelNotFound
	}

	reporter.nodeCancel()
	reporter.isRunning = false
	reporter.KaiaHelper.Close()
	<-reporter.nodeCtx.Done()
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
