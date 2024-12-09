package reporter

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"

	"github.com/rs/zerolog/log"
)

const maxTxSubmissionRetries = 3

func NewReporter(ctx context.Context, opts ...ReporterOption) (*Reporter, error) {
	config := &ReporterConfig{
		JobType: ReportJob,
	}

	for _, opt := range opts {
		opt(config)
	}

	if len(config.Configs) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no submission pairs to make new reporter")
		return nil, errorSentinel.ErrReporterEmptyConfigs
	}

	groupInterval := time.Duration(config.Interval) * time.Millisecond

	deviationThreshold := GetDeviationThreshold(groupInterval)

	reporter := &Reporter{
		contractAddress:        config.ContractAddress,
		SubmissionInterval:     groupInterval,
		CachedWhitelist:        config.CachedWhitelist,
		deviationThreshold:     deviationThreshold,
		KaiaHelper:             config.KaiaHelper,
		LatestDataMap:          config.LatestDataMap,
		LatestSubmittedDataMap: config.LatestSubmittedDataMap,
	}

	reporter.Pairs = make([]string, 0, len(config.Configs))
	for _, sa := range config.Configs {
		reporter.Pairs = append(reporter.Pairs, sa.Name)
	}

	if config.JobType == ReportJob {
		reporter.Job = func() error {
			return reporter.regularReporterJob(ctx)
		}
	} else {
		reporter.Job = func() error {
			return reporter.deviationJob(ctx)
		}
	}

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	log.Info().Msgf("Reporter ticker starting with interval: %v", r.SubmissionInterval)
	ticker := time.NewTicker(r.SubmissionInterval)

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("Player", "Reporter").Msg("context done, stopping reporter")
			return
		case <-ticker.C:
			go func() {
				err := r.Job()
				if err != nil {
					log.Error().Str("Player", "Reporter").Err(err).Msg("ReporterJob")
				}
			}()
		}
	}
}

func (r *Reporter) regularReporterJob(ctx context.Context) error {
	pairsMap, err := GetLatestDataRest(ctx, r.Pairs)
	if err != nil {
		return err
	}

	err = r.report(ctx, pairsMap)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reporter) deviationJob(ctx context.Context) error {
	deviatingAggregates := GetDeviatingAggregates(r.LatestSubmittedDataMap, r.LatestDataMap, r.deviationThreshold)
	if len(deviatingAggregates) == 0 {
		return nil
	}
	log.Debug().Str("Player", "Reporter").Msgf("deviating aggregates found: %v", deviatingAggregates)

	err := r.report(ctx, deviatingAggregates)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reporter) report(ctx context.Context, pairs map[string]SubmissionData) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte
	var submittedPairs []string

	for pair, submissionData := range pairs {
		feedHashes = append(feedHashes, submissionData.FeedHash)
		values = append(values, big.NewInt(submissionData.Value))
		timestamps = append(timestamps, big.NewInt(submissionData.AggregateTime))
		proofs = append(proofs, submissionData.Proof)
		submittedPairs = append(submittedPairs, pair)
	}

	dataLen := len(feedHashes)
	wg := sync.WaitGroup{}

	errorsChan := make(chan error, 5)
	for start := 0; start < dataLen; start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, dataLen)

		batchFeedHashes := feedHashes[start:end]
		batchValues := values[start:end]
		batchTimestamps := timestamps[start:end]
		batchProofs := proofs[start:end]
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := r.KaiaHelper.SubmitDelegatedFallbackDirect(ctx, r.contractAddress, SUBMIT_WITH_PROOFS, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
			if err != nil {
				errorsChan <- err
			}
		}()
	}
	wg.Wait()
	close(errorsChan)

	shouldRefreshNonce := false

	tmp := []error{}
	for err := range errorsChan {
		tmp = append(tmp, err)
		if utils.IsNonceError(err) || errors.Is(err, context.DeadlineExceeded) {
			log.Debug().Err(err).Str("Player", "Reporter").Msg("should refresh nonce")
			shouldRefreshNonce = true
		}
	}

	for i, pair := range submittedPairs {
		r.LatestSubmittedDataMap.Store(pair, values[i].Int64())
	}

	if shouldRefreshNonce {
		log.Debug().Str("Player", "Reporter").Msg("refreshing nonce pool")
		return r.KaiaHelper.FlushNoncePool(ctx)
	}

	if len(tmp) > 0 {
		return mergeErrors(tmp)
	}

	log.Debug().Str("Player", "Reporter").Msgf("reporting done for reporter with interval: %v", r.SubmissionInterval)

	return nil
}

func mergeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}
	var errMsg string
	for _, err := range errs {
		if err != nil {
			errMsg += err.Error() + "; "
		}
	}
	return errors.New(errMsg)
}
