package reporter

import (
	"context"
	"encoding/json"
	"errors"
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

	close(timestampsChan)
	close(valuesChan)
	close(feedHashesChan)
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

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {

	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting")

	if !ValidateAggregateTimestampValues(aggregates) {
		log.Error().Str("Player", "Reporter").Msg("ValidateAggregateTimestampValues, zero timestamp exists")
		return errorSentinel.ErrReporterValidateAggregateTimestampValues
	}

	proofMap, err := GetProofsAsMap(ctx, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return err
	}
	log.Debug().Str("Player", "Reporter").Int("proofs", len(proofMap)).Msg("proof map generated")

	orderedProofMap, err := r.orderProofs(ctx, proofMap, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("orderProofs")
		return err
	}

	log.Debug().Str("Player", "Reporter").Int("orderedProofs", len(orderedProofMap)).Msg("ordered proof map generated")

	err = UpdateProofs(ctx, aggregates, orderedProofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("updateProofs")
		return err
	}
	log.Debug().Str("Player", "Reporter").Msg("proofs updated to db, reporting with proofs")
	return r.reportWithProofs(ctx, aggregates, orderedProofMap)
}

func (r *Reporter) orderProof(ctx context.Context, proof []byte, aggregate GlobalAggregate) ([]byte, error) {
	proof = RemoveDuplicateProof(proof)
	hash := chainUtils.Value2HashForSign(aggregate.Value, aggregate.Timestamp.Unix(), r.SubmissionPairs[aggregate.ConfigID].Name)
	proofChunks, err := SplitProofToChunk(proof)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to split proof")
		return nil, err
	}

	signers, err := GetSignerListFromProofs(hash, proofChunks)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get signers from proof")
		return nil, err
	}

	err = CheckForNonWhitelistedSigners(signers, r.CachedWhitelist)
	if err != nil {
		log.Warn().Str("Player", "Reporter").Err(err).Msg("non-whitelisted signers in proof, reloading whitelist")
		reloadedWhitelist, contractReadErr := ReadOnchainWhitelist(ctx, r.KaiaHelper, r.contractAddress, GET_ONCHAIN_WHITELIST)
		if contractReadErr != nil {
			log.Error().Str("Player", "Reporter").Err(contractReadErr).Msg("failed to reload whitelist")
			return nil, contractReadErr
		}
		r.CachedWhitelist = reloadedWhitelist
	}

	signerMap := GetSignerMap(signers, proofChunks)
	orderedProof, err := OrderProof(signerMap, r.CachedWhitelist)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to order proof")
		return nil, err
	}
	return orderedProof, nil
}

func (r *Reporter) orderProofs(ctx context.Context, proofMap map[int32][]byte, aggregates []GlobalAggregate) (map[int32][]byte, error) {
	orderedProofMap := make(map[int32][]byte)
	for _, agg := range aggregates {
		proof, ok := proofMap[agg.ConfigID]
		if !ok {
			log.Error().Str("Player", "Reporter").Msg("proof not found")
			return nil, errorSentinel.ErrReporterProofNotFound
		}

		orderedProof, err := r.orderProof(ctx, proof, agg)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("orderProof")
			return nil, err
		}

		orderedProofMap[agg.ConfigID] = orderedProof
	}

	return orderedProofMap, nil
}

func (r *Reporter) resignLeader() {
	r.Raft.ResignLeader()
}

func (r *Reporter) handleCustomMessage(ctx context.Context, msg raft.Message) error {
	switch msg.Type {
	case SubmissionMsg:
		return r.HandleSubmissionMessage(ctx, msg)
	default:
		return errorSentinel.ErrReporterUnknownMessageType
	}
}

func (r *Reporter) reportWithProofs(ctx context.Context, aggregates []GlobalAggregate, proofMap map[int32][]byte) error {
	if r.KaiaHelper == nil {
		return errorSentinel.ErrReporterKaiaHelperNotFound
	}
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting with proofs")

	feedHashes, values, timestamps, proofs, err := MakeContractArgsWithProofs(aggregates, r.SubmissionPairs, proofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithProofs")
		return err
	}
	log.Debug().Str("Player", "Reporter").Int("proofs", len(proofs)).Msg("contract arguements generated")

	return r.splitReport(ctx, feedHashes, values, timestamps, proofs)
}

func (r *Reporter) SetKaiaHelper(ctx context.Context) error {
	if r.KaiaHelper != nil {
		r.KaiaHelper.Close()
	}
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create kaia helper")
		return err
	}
	r.KaiaHelper = kaiaHelper
	return nil
}

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

	err = r.PublishSubmissionMessage(deviatingAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("PublishSubmissionMessage")
		return err
	}

	for _, agg := range deviatingAggregates {
		pair := r.SubmissionPairs[agg.ConfigID]
		pair.LastSubmission = agg.Round
		r.SubmissionPairs[agg.ConfigID] = pair
	}

	log.Info().Int("deviations", len(deviatingAggregates)).Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting deviation done")
	return nil
}

func (r *Reporter) PublishSubmissionMessage(submissions []GlobalAggregate) error {
	submissionMessage := SubmissionMessage{Submissions: submissions}
	marshalledSubmissionMessage, err := json.Marshal(submissionMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     SubmissionMsg,
		SentFrom: r.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledSubmissionMessage),
	}

	return r.Raft.PublishMessage(message)
}

func (r *Reporter) HandleSubmissionMessage(ctx context.Context, msg raft.Message) error {
	var submissionMessage SubmissionMessage
	err := json.Unmarshal(msg.Data, &submissionMessage)
	if err != nil {
		return err
	}

	err = StoreLastSubmission(ctx, submissionMessage.Submissions)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("storeLastSubmission")
		return err
	}

	return nil
}

func (r *Reporter) splitReport(ctx context.Context, feedHashes [][32]byte, values []*big.Int, timestamps []*big.Int, proofs [][]byte) error {
	errs := []error{}
	for start := 0; start < len(feedHashes); start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, len(feedHashes))

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
