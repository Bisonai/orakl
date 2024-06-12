package reporter

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	chainUtils "bisonai.com/orakl/node/pkg/chain/utils"
	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"bisonai.com/orakl/node/pkg/raft"
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

	topicString := TOPIC_STRING + "-"
	if config.JobType == DeviationJob {
		topicString += "deviation-" + strconv.Itoa(config.Interval)
	} else {
		topicString += strconv.Itoa(config.Interval)
	}

	groupInterval := time.Duration(config.Interval) * time.Millisecond

	topic, err := config.Ps.Join(topicString)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	raft := raft.NewRaftNode(config.Host, config.Ps, topic, MESSAGE_BUFFER, groupInterval)
	reporter := &Reporter{
		Raft:               raft,
		contractAddress:    config.ContractAddress,
		SubmissionInterval: groupInterval,
		CachedWhitelist:    config.CachedWhitelist,
	}
	reporter.SubmissionPairs = make(map[int32]SubmissionPair)
	for _, sa := range config.Configs {
		reporter.SubmissionPairs[sa.ID] = SubmissionPair{LastSubmission: 0, Name: sa.Name}
	}
	reporter.Raft.HandleCustomMessage = reporter.handleCustomMessage
	if config.JobType == DeviationJob {
		reporter.Raft.LeaderJob = reporter.deviationJob
	} else {
		reporter.Raft.LeaderJob = reporter.leaderJob
	}

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	r.Raft.Run(ctx)
}

func (r *Reporter) leaderJob() error {
	start := time.Now()
	log.Info().Str("Player", "Reporter").Time("start", start).Msg("reporter job")
	r.Raft.IncreaseTerm()
	ctx := context.Background()

	loadLatestGlobalAggregateStart := time.Now()
	aggregates, err := GetLatestGlobalAggregates(ctx, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetLatestGlobalAggregates")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(loadLatestGlobalAggregateStart).String()).Msg("loaded latest global aggregates")

	filterValidAggregateStart := time.Now()
	validAggregates := FilterInvalidAggregates(aggregates, r.SubmissionPairs)
	if len(validAggregates) == 0 {
		log.Warn().Str("Player", "Reporter").Msg("no valid aggregates to report")
		return nil
	}
	log.Debug().Str("Player", "Reporter").Int("validAggregates", len(validAggregates)).Msg("valid aggregates")
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(filterValidAggregateStart).String()).Msg("filtered valid aggregates")

	reportStart := time.Now()
	reportJob := func() error {
		err = r.report(ctx, validAggregates)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("report")
			return err
		}
		return nil
	}

	err = retrier.Retry(
		reportJob,
		MAX_RETRY,
		INITIAL_FAILURE_TIMEOUT,
		MAX_RETRY_DELAY,
	)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report, resigning from leader")
		r.resignLeader()
		return errorSentinel.ErrReporterReportFailed
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(reportStart).String()).Msg("reported from reporter leader job")

	publishSubmissionMessageStart := time.Now()
	err = r.PublishSubmissionMessage(validAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("PublishSubmissionMessage")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(publishSubmissionMessageStart).String()).Msg("published submission message")

	updateValidAggregatesStart := time.Now()
	for _, agg := range validAggregates {
		pair := r.SubmissionPairs[agg.ConfigID]
		pair.LastSubmission = agg.Round
		r.SubmissionPairs[agg.ConfigID] = pair
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(updateValidAggregatesStart).String()).Msg("updated valid aggregates")
	log.Info().Int("validAggregates", len(validAggregates)).Str("Player", "Reporter").Str("Duration", time.Since(start).String()).Msg("reporting done")

	return nil
}

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {
	startPrepareProof := time.Now()
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting")

	startPrepareProofMap := time.Now()
	proofMap, err := GetProofsAsMap(ctx, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(startPrepareProofMap).String()).Msg("prepared proof map")
	log.Debug().Str("Player", "Reporter").Int("proofs", len(proofMap)).Msg("proof map generated")

	startOrderProofMap := time.Now()
	orderedProofMap, err := r.orderProofs(ctx, proofMap, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("orderProofs")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(startOrderProofMap).String()).Msg("ordered proof map")
	log.Debug().Str("Player", "Reporter").Int("orderedProofs", len(orderedProofMap)).Msg("ordered proof map generated")

	startUpdateProofMap := time.Now()
	err = UpdateProofs(ctx, aggregates, orderedProofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("updateProofs")
		return err
	}
	log.Debug().Str("Player", "Reporter").Msg("proofs updated to db, reporting with proofs")

	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(startUpdateProofMap).String()).Msg("proofs updated")
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(startPrepareProof).String()).Msg("func report()")
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
		reloadedWhitelist, contractReadErr := ReadOnchainWhitelist(ctx, r.KlaytnHelper, r.contractAddress, GET_ONCHAIN_WHITELIST)
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
	now := time.Now()
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting with proofs")
	if r.KlaytnHelper == nil {
		return errorSentinel.ErrReporterKlaytnHelperNotFound
	}

	startMakingContractArgs := time.Now()
	feedHashes, values, timestamps, proofs, err := MakeContractArgsWithProofs(aggregates, r.SubmissionPairs, proofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithProofs")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(startMakingContractArgs).String()).Msg("contract arguements generated")
	log.Debug().Str("Player", "Reporter").Int("proofs", len(proofs)).Msg("contract arguements generated")

	startReport := time.Now()
	err = r.reportDelegated(ctx, SUBMIT_WITH_PROOFS, feedHashes, values, timestamps, proofs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, SUBMIT_WITH_PROOFS, feedHashes, values, timestamps, proofs)
	}
	log.Info().Str("Player", "Reporter").Int("aggregates", len(values)).Int("proofs", len(proofs)).Str("Duration", time.Since(startReport).String()).Msg("delegated reported")
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(now).String()).Msg("submitted with proofs")
	return nil
}

func (r *Reporter) reportDirect(ctx context.Context, functionString string, args ...interface{}) error {
	rawTx, err := r.KlaytnHelper.MakeDirectTx(ctx, r.contractAddress, functionString, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeDirectTx")
		return err
	}

	return r.KlaytnHelper.SubmitRawTx(ctx, rawTx)
}

func (r *Reporter) reportDelegated(ctx context.Context, functionString string, args ...interface{}) error {
	log.Debug().Str("Player", "Reporter").Msg("reporting delegated")
	rawTx, err := r.KlaytnHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, functionString, GAS_MULTIPLIER, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}

	log.Debug().Str("Player", "Reporter").Str("RawTx", rawTx.String()).Msg("delegated raw tx generated")
	delegatorRequestStart := time.Now()
	signedTx, err := r.KlaytnHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetSignedFromDelegator")
		return err
	}
	log.Info().Str("Player", "Reporter").Str("Duration", time.Since(delegatorRequestStart).String()).Msg("delegator signed tx generated")
	log.Debug().Str("Player", "Reporter").Str("signedTx", signedTx.String()).Msg("signed tx generated, submitting raw tx")

	return r.KlaytnHelper.SubmitRawTx(ctx, signedTx)
}

func (r *Reporter) SetKlaytnHelper(ctx context.Context) error {
	if r.KlaytnHelper != nil {
		r.KlaytnHelper.Close()
	}
	klaytnHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create klaytn helper")
		return err
	}
	r.KlaytnHelper = klaytnHelper
	return nil
}

func (r *Reporter) deviationJob() error {
	start := time.Now()
	log.Info().Str("Player", "Reporter").Time("start", start).Msg("reporter deviation job")
	r.Raft.IncreaseTerm()
	ctx := context.Background()

	lastSubmissions, err := GetLastSubmission(ctx, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLastSubmission")
		return err
	}
	if len(lastSubmissions) == 0 {
		log.Warn().Str("Player", "Reporter").Msg("no last submissions")
		return nil
	}

	lastAggregates, err := GetLatestGlobalAggregates(ctx, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregates")
		return err
	}
	if len(lastAggregates) == 0 {
		log.Warn().Str("Player", "Reporter").Msg("no last aggregates")
		return nil
	}

	deviatingAggregates := GetDeviatingAggregates(lastSubmissions, lastAggregates)
	if len(deviatingAggregates) == 0 {
		return nil
	}

	reportJob := func() error {
		err = r.report(ctx, deviatingAggregates)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("DeviationReport")
			return err
		}
		return nil
	}

	err = retrier.Retry(
		reportJob,
		MAX_RETRY,
		INITIAL_FAILURE_TIMEOUT,
		MAX_RETRY_DELAY,
	)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report deviation, resigning from leader")
		r.resignLeader()
		return errorSentinel.ErrReporterDeviationReportFail
	}

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

	log.Info().Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting deviation done")
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
