package reporter

import (
	"context"
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

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("Player", "Reporter").Msg("context done, stopping reporter")
			return
		case <-ticker.C:
			err := r.report(ctx)
			if err != nil {
				log.Error().Str("Player", "Reporter").Err(err).Msg("reporting failed")
			}
		}
	}
}

func (r *Reporter) report(ctx context.Context) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte

	for _, pair := range r.SubmissionPairs {
		if value, ok := r.LatestData.Load(pair.Name); ok {
			submissionData, err := processDalWsRawData(value)
			if err != nil {
				log.Error().Str("Player", "Reporter").Err(err).Msg("failed to process dal ws raw data")
				continue
			}

			feedHashes = append(feedHashes, submissionData.feedHash)
			values = append(values, big.NewInt(submissionData.value))
			timestamps = append(timestamps, big.NewInt(submissionData.aggregateTime))
			proofs = append(proofs, submissionData.proof)
		} else {
			log.Error().Str("Player", "Reporter").Msgf("latest data for pair %s not found", pair.Name)
		}
	}

	dataLen := len(feedHashes)
	for start := 0; start < dataLen; start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, dataLen)

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
	mapData, mapDataOk := data.(map[string]interface{})
	if !mapDataOk {
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData := SubmissionData{}

	intValue, valueErr := strconv.ParseInt(mapData["value"].(string), 10, 64)
	if valueErr != nil {
		log.Error().Str("Player", "Reporter").Err(valueErr).Msg("failed to parse value")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.value = intValue

	timestampValue, timestampErr := strconv.ParseInt(mapData["aggregateTime"].(string), 10, 64)
	if timestampErr != nil {
		log.Error().Str("Player", "Reporter").Err(timestampErr).Msg("failed to parse timestamp")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.aggregateTime = timestampValue

	var feedHash [32]byte
	interfaceFeedHash, interfaceFeedHashOk := mapData["feedHash"].([]interface{})
	if !interfaceFeedHashOk {
		log.Error().Str("Player", "Reporter").Msg("failed to convert feed hash to interface")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed

	}
	if len(interfaceFeedHash) == 32 {
		for i, v := range interfaceFeedHash {
			floatVal, ok := v.(float64)
			if !ok {
				log.Error().Str("Player", "Reporter").Msg("failed to convert feed hash value to float")
				return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
			}
			feedHash[i] = byte(floatVal)
		}
	} else {
		log.Error().Str("Player", "Reporter").Msg("failed to convert feed hash to byte, length not 32")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.feedHash = feedHash

	proof, ok := mapData["proof"].(string)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("failed to convert proof to string")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.proof = []byte(proof)

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
