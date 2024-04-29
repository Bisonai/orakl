package reporter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
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
		return errors.New("SUBMISSION_PROXY_CONTRACT not set")
	}

	tmpChainHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create chain helper")
		return err
	}
	defer tmpChainHelper.Close()

	cachedWhitelist, err := ReadOnchainWhitelist(ctx, tmpChainHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get whitelist, starting with empty whitelist")
		cachedWhitelist = []common.Address{}
	}

	reporterConfigs, err := getReporterConfigs(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get reporter configs")
		return err
	}

	groupedReporterConfigs := groupReporterConfigsByIntervals(reporterConfigs)
	for groupInterval, configs := range groupedReporterConfigs {
		reporter, errNewReporter := NewReporter(ctx, h, ps, configs, groupInterval, contractAddress, cachedWhitelist)
		if errNewReporter != nil {
			log.Error().Str("Player", "Reporter").Err(errNewReporter).Msg("failed to set reporter")
			continue
		}
		a.Reporters = append(a.Reporters, reporter)
	}

	if len(a.Reporters) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no reporters set")
		return errors.New("no reporters set")
	}

	deviationReporter, err := NewDeviationReporter(ctx, h, ps, reporterConfigs, contractAddress, cachedWhitelist)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create deviation reporter")
		return err
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
		return fmt.Errorf("errors occurred while stopping reporters: %v", errs)
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
		return fmt.Errorf(strings.Join(errs, "; "))
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
		return fmt.Errorf(strings.Join(errs, "; "))
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
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
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
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
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
			bus.HandleMessageError(errors.New("non-admin"), msg, "reporter received message from non-admin")
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
	return nil, errors.New("reporter not found")
}

func startReporter(ctx context.Context, reporter *Reporter) error {
	if reporter.isRunning {
		log.Debug().Str("Player", "Reporter").Msg("reporter already running")
		return errors.New("reporter already running")
	}

	err := reporter.SetKlaytnHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to set klaytn helper")
		return err
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
		return errors.New("reporter cancel function not found")
	}

	reporter.nodeCancel()
	reporter.isRunning = false
	reporter.KlaytnHelper.Close()
	<-reporter.nodeCtx.Done()
	return nil
}

func getReporterConfigs(ctx context.Context) ([]ReporterConfig, error) {
	reporterConfigs, err := db.QueryRows[ReporterConfig](ctx, GET_REPORTER_CONFIGS, nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to load reporter configs")
		return nil, err
	}
	return reporterConfigs, nil
}

func groupReporterConfigsByIntervals(reporterConfigs []ReporterConfig) map[int][]ReporterConfig {
	grouped := make(map[int][]ReporterConfig)
	for _, sa := range reporterConfigs {
		var interval = 5000
		if sa.SubmitInterval != nil && *sa.SubmitInterval > 0 {
			interval = *sa.SubmitInterval
		}
		grouped[interval] = append(grouped[interval], sa)
	}
	return grouped
}
