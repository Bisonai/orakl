package reporter

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"sync"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"github.com/rs/zerolog/log"
)

var mu sync.Mutex

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
	// ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("Player", "Reporter").Msg("context done, stopping reporter")
			return
		case <-ticker.C:
			go func() {
				err := r.report(ctx)
				if err != nil {
					log.Error().Str("Player", "Reporter").Err(err).Msg("reporting failed")
				}
			}()
		}
	}
}

func (r *Reporter) report(ctx context.Context) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte

	feedHashesChan := make(chan [32]byte, len(r.SubmissionPairs))
	valuesChan := make(chan *big.Int, len(r.SubmissionPairs))
	timestampsChan := make(chan *big.Int, len(r.SubmissionPairs))
	proofsChan := make(chan []byte, len(r.SubmissionPairs))

	wg := sync.WaitGroup{}

	for _, pair := range r.SubmissionPairs {
		wg.Add(1)
		go func(pair SubmissionPair) {
			defer wg.Done()
			if value, ok := r.LatestData.Load(pair.Name); ok {
				submissionData, err := processDalWsRawData(value)
				if err != nil {
					log.Error().Str("Player", "Reporter").Err(err).Msg("failed to process dal ws raw data")
					return
				}

				feedHashesChan <- submissionData.FeedHash
				valuesChan <- big.NewInt(submissionData.Value)
				timestampsChan <- big.NewInt(submissionData.AggregateTime)
				proofsChan <- submissionData.Proof
			} else {
				log.Error().Str("Player", "Reporter").Msgf("latest data for pair %s not found", pair.Name)
			}
		}(pair)
	}

	wg.Wait()

	close(feedHashesChan)
	close(valuesChan)
	close(timestampsChan)
	close(proofsChan)

	for feedHash := range feedHashesChan {
		feedHashes = append(feedHashes, feedHash)
		values = append(values, <-valuesChan)
		timestamps = append(timestamps, <-timestampsChan)
		proofs = append(proofs, <-proofsChan)
	}

	dataLen := len(feedHashes)
	for start := 0; start < dataLen; start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, dataLen-1)

		batchFeedHashes := feedHashes[start:end]
		batchValues := values[start:end]
		batchTimestamps := timestamps[start:end]
		batchProofs := proofs[start:end]

		err := r.reportDelegated(ctx, SUBMIT_WITH_PROOFS, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
		if err != nil {
			err = r.reportDirect(ctx, SUBMIT_WITH_PROOFS, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
			if err != nil {
				log.Error().Str("Player", "Reporter").Err(err).Msg("report")
			}
			log.Error().Str("Player", "Reporter").Err(err).Msg("report")
		}
	}
	return nil
}

func (r *Reporter) reportDirect(ctx context.Context, functionString string, args ...interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	rawTx, err := r.KaiaHelper.MakeDirectTx(ctx, r.contractAddress, functionString, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeDirectTx")
		return err
	}

	return r.KaiaHelper.SubmitRawTx(ctx, rawTx)
}

func (r *Reporter) reportDelegated(ctx context.Context, functionString string, args ...interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	log.Debug().Str("Player", "Reporter").Msg("reporting delegated")
	rawTx, err := r.KaiaHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, functionString, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}

	log.Debug().Str("Player", "Reporter").Str("RawTx", rawTx.String()).Msg("delegated raw tx generated")
	signedTx, err := r.KaiaHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetSignedFromDelegator")
		return err
	}
	log.Debug().Str("Player", "Reporter").Str("signedTx", signedTx.String()).Msg("signed tx generated, submitting raw tx")

	return r.KaiaHelper.SubmitRawTx(ctx, signedTx)
}

func processDalWsRawData(data any) (SubmissionData, error) {
	rawSubmissionData := RawSubmissionData{}

	jsonMarshalData, jsonMarshalDataErr := json.Marshal(data)
	if jsonMarshalDataErr != nil {
		log.Error().Str("Player", "Reporter").Err(jsonMarshalDataErr).Msg("failed to marshal data")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}

	jsonUnmarshalDataErr := json.Unmarshal(jsonMarshalData, &rawSubmissionData)
	if jsonUnmarshalDataErr != nil {
		log.Error().Str("Player", "Reporter").Err(jsonUnmarshalDataErr).Msg("failed to unmarshal data")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}

	submissionData := SubmissionData{
		FeedHash: rawSubmissionData.FeedHash,
		Proof:    rawSubmissionData.Proof,
	}

	value, valueErr := strconv.ParseInt(rawSubmissionData.Value, 10, 64)
	if valueErr != nil {
		log.Error().Str("Player", "Reporter").Err(valueErr).Msg("failed to parse value")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.Value = value

	timestampValue, timestampErr := strconv.ParseInt(rawSubmissionData.AggregateTime, 10, 64)
	if timestampErr != nil {
		log.Error().Str("Player", "Reporter").Err(timestampErr).Msg("failed to parse timestamp")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.AggregateTime = timestampValue

	return submissionData, nil
}

// func (r *Reporter) deviationJob() error {
// 	start := time.Now()
// 	log.Info().Str("Player", "Reporter").Time("start", start).Msg("reporter deviation job")
// 	ctx := context.Background()

// 	lastSubmissions, err := GetLastSubmission(ctx, r.SubmissionPairs)
// 	if err != nil {
// 		log.Error().Str("Player", "Reporter").Err(err).Msg("getLastSubmission")
// 		return err
// 	}
// 	if len(lastSubmissions) == 0 {
// 		log.Warn().Str("Player", "Reporter").Msg("no last submissions")
// 		return nil
// 	}

// 	lastAggregates, err := GetLatestGlobalAggregates(ctx, r.SubmissionPairs)
// 	if err != nil {
// 		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregates")
// 		return err
// 	}
// 	if len(lastAggregates) == 0 {
// 		log.Warn().Str("Player", "Reporter").Msg("no last aggregates")
// 		return nil
// 	}

// 	deviatingAggregates := GetDeviatingAggregates(lastSubmissions, lastAggregates, r.deviationThreshold)
// 	if len(deviatingAggregates) == 0 {
// 		return nil
// 	}

// 	reportJob := func() error {
// 		err = r.report(ctx, deviatingAggregates)
// 		if err != nil {
// 			log.Error().Str("Player", "Reporter").Err(err).Msg("DeviationReport")
// 			return err
// 		}
// 		return nil
// 	}

// 	err = retrier.Retry(
// 		reportJob,
// 		MAX_RETRY,
// 		INITIAL_FAILURE_TIMEOUT,
// 		MAX_RETRY_DELAY,
// 	)
// 	if err != nil {
// 		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report deviation, resigning from leader")
// 		return errorSentinel.ErrReporterDeviationReportFail
// 	}

// 	for _, agg := range deviatingAggregates {
// 		pair := r.SubmissionPairs[agg.ConfigID]
// 		pair.LastSubmission = agg.Round
// 		r.SubmissionPairs[agg.ConfigID] = pair
// 	}

// 	log.Info().Int("deviations", len(deviatingAggregates)).Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting deviation done")
// 	return nil
// }
