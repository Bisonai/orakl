package reporter

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/retrier"

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
		contractAddress:    config.ContractAddress,
		SubmissionInterval: groupInterval,
		CachedWhitelist:    config.CachedWhitelist,
		deviationThreshold: deviationThreshold,
		JobType:            config.JobType,
	}

	reporter.SubmissionPairs = make(map[int32]SubmissionPair)
	for _, sa := range config.Configs {
		reporter.SubmissionPairs[sa.ID] = SubmissionPair{LastSubmission: 0, Name: sa.Name}
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
			if r.JobType == ReportJob {
				go func() {
					err := r.report(ctx, r.SubmissionPairs)
					if err != nil {
						log.Error().Str("Player", "Reporter").Err(err).Msg("reporting failed")
					}
				}()
			} else {
				go func() {
					err := r.deviationJob(ctx)
					if err != nil {
						log.Error().Str("Player", "Reporter").Err(err).Msg("deviation job failed")
					}
				}()
			}
		}
	}
}

func (r *Reporter) report(ctx context.Context, submissionPairs map[int32]SubmissionPair) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte
	var configIds []int32

	feedHashesChan := make(chan [32]byte, len(submissionPairs))
	valuesChan := make(chan *big.Int, len(submissionPairs))
	timestampsChan := make(chan *big.Int, len(submissionPairs))
	proofsChan := make(chan []byte, len(submissionPairs))
	configIdChan := make(chan int32, len(submissionPairs))

	wg := sync.WaitGroup{}

	for id, pair := range submissionPairs {
		wg.Add(1)
		go func(id int32, pair SubmissionPair) {
			defer wg.Done()
			if value, ok := r.LatestData.Load(pair.Name); ok {
				submissionData, submissionDataOk := value.(SubmissionData)
				if !submissionDataOk {
					log.Error().Str("Player", "Reporter").Msg("latest data not found")
					return
				}

				feedHashesChan <- submissionData.FeedHash
				valuesChan <- big.NewInt(submissionData.Value)
				timestampsChan <- big.NewInt(submissionData.AggregateTime)
				proofsChan <- submissionData.Proof
				configIdChan <- id
			} else {
				log.Error().Str("Player", "Reporter").Msgf("latest data for pair %s not found", pair.Name)
			}
		}(id, pair)
	}

	wg.Wait()

	close(timestampsChan)
	close(valuesChan)
	close(feedHashesChan)
	close(proofsChan)
	close(configIdChan)

	for feedHash := range feedHashesChan {
		feedHashes = append(feedHashes, feedHash)
		values = append(values, <-valuesChan)
		timestamps = append(timestamps, <-timestampsChan)
		proofs = append(proofs, <-proofsChan)
		configIds = append(configIds, <-configIdChan)
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

	// assumption: values[i] will always correspond to configIds[i] (theoretically doesn't have to be the case)
	for i, configId := range configIds {
		pair := r.SubmissionPairs[configId]
		pair.LastSubmission = values[i].Int64()
		r.SubmissionPairs[configId] = pair
	}

	if len(errs) > 0 {
		return mergeErrors(errs)
	}

	log.Debug().Str("Player", "Reporter").Msgf("reporting done for reporter with interval: %v", r.SubmissionInterval)

	return nil
}

func (r *Reporter) deviationJob(ctx context.Context) error {
	deviatingAggregates := GetDeviatingAggregates(r.SubmissionPairs, r.LatestData, r.deviationThreshold)
	if len(deviatingAggregates) == 0 {
		log.Debug().Str("Player", "Reporter").Msg("no deviating aggregates found")
		return nil
	}
	log.Debug().Str("Player", "Reporter").Msgf("deviating aggregates found: %v", deviatingAggregates)

	reportJob := func() error {
		err := r.report(ctx, deviatingAggregates)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("DeviationReport")
			return err
		}
		return nil
	}

	err := retrier.Retry(
		reportJob,
		MAX_RETRY,
		INITIAL_FAILURE_TIMEOUT,
		MAX_RETRY_DELAY,
	)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report deviation, resigning from leader")
		return errorSentinel.ErrReporterDeviationReportFail
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
