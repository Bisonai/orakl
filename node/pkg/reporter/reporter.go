package reporter

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"github.com/rs/zerolog/log"
)

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
		contractAddress:     config.ContractAddress,
		SubmissionInterval:  groupInterval,
		CachedWhitelist:     config.CachedWhitelist,
		deviationThreshold:  deviationThreshold,
		KaiaHelper:          config.KaiaHelper,
		LatestData:          config.LatestData,
		LatestSubmittedData: config.LatestSubmittedData,
	}

	reporter.Pairs = make([]string, 0, len(config.Configs))
	for _, sa := range config.Configs {
		reporter.Pairs = append(reporter.Pairs, sa.Name)
	}

	if config.JobType == ReportJob {
		reporter.Job = func() error {
			return reporter.report(ctx, reporter.Pairs)
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

func (r *Reporter) report(ctx context.Context, pairs []string) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte
	var submittedPairs []string

	feedHashesChan := make(chan [32]byte, len(pairs))
	valuesChan := make(chan *big.Int, len(pairs))
	timestampsChan := make(chan *big.Int, len(pairs))
	proofsChan := make(chan []byte, len(pairs))
	submittedPairsChan := make(chan string, len(pairs))

	wg := sync.WaitGroup{}

	for _, pair := range pairs {
		wg.Add(1)
		go func(pair string) {
			defer wg.Done()
			submissionData, ok := GetLatestData(r.LatestData, pair)
			if !ok {
				log.Error().Str("Player", "Reporter").Msgf("latest data for pair %s not found", pair)
				return
			}

			feedHashesChan <- submissionData.FeedHash
			valuesChan <- big.NewInt(submissionData.Value)
			timestampsChan <- big.NewInt(submissionData.AggregateTime)
			proofsChan <- submissionData.Proof
			submittedPairsChan <- pair
		}(pair)
	}

	wg.Wait()

	close(timestampsChan)
	close(valuesChan)
	close(feedHashesChan)
	close(proofsChan)
	close(submittedPairsChan)

	for feedHash := range feedHashesChan {
		feedHashes = append(feedHashes, feedHash)
		values = append(values, <-valuesChan)
		timestamps = append(timestamps, <-timestampsChan)
		proofs = append(proofs, <-proofsChan)
		submittedPairs = append(submittedPairs, <-submittedPairsChan)
	}

	errs := []error{}
	dataLen := len(feedHashes)
	for start := 0; start < dataLen; start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, dataLen-1)

		batchFeedHashes := feedHashes[start:end]
		batchValues := values[start:end]
		batchTimestamps := timestamps[start:end]
		batchProofs := proofs[start:end]

		err := r.KaiaHelper.SubmitDelegatedFallbackDirect(ctx, r.contractAddress, SUBMIT_WITH_PROOFS, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("splitReport")
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return mergeErrors(errs)
	}

	for i, pair := range submittedPairs {
		r.LatestSubmittedData.Store(pair, values[i].Int64())
	}

	log.Debug().Str("Player", "Reporter").Msgf("reporting done for reporter with interval: %v", r.SubmissionInterval)

	return nil
}

func (r *Reporter) deviationJob(ctx context.Context) error {
	deviatingAggregates := GetDeviatingAggregates(r.LatestSubmittedData, r.LatestData, r.deviationThreshold)
	if len(deviatingAggregates) == 0 {
		log.Debug().Str("Player", "Reporter").Msg("no deviating aggregates found")
		return nil
	}
	log.Debug().Str("Player", "Reporter").Msgf("deviating aggregates found: %v", deviatingAggregates)

	err := r.report(ctx, deviatingAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("DeviationReport")
		return err
	}
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
