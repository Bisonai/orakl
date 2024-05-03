package reporter

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	chain_utils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/retrier"

	"github.com/klaytn/klaytn/common"
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
		log.Error().Str("Player", "Reporter").Err(errors.New("no submission pairs")).Msg("no submission pairs to make new reporter")
		return nil, errors.New("no submission pairs")
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
		reporter.SubmissionPairs[sa.ID] = SubmissionPair{LastSubmission: 0, Address: common.HexToAddress(sa.Address)}
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
	r.Raft.IncreaseTerm()
	ctx := context.Background()

	aggregates, err := GetLatestGlobalAggregates(ctx, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetLatestGlobalAggregates")
		return err
	}

	validAggregates := FilterInvalidAggregates(aggregates, r.SubmissionPairs)
	if len(validAggregates) == 0 {
		log.Warn().Str("Player", "Reporter").Msg("no valid aggregates to report")
		return nil
	}
	log.Debug().Str("Player", "Reporter").Int("validAggregates", len(validAggregates)).Msg("valid aggregates")

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
		return errors.New("failed to report")
	}

	err = r.PublishSubmissionMessage(validAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("PublishSubmissionMessage")
		return err
	}

	for _, agg := range validAggregates {
		pair := r.SubmissionPairs[agg.ConfigID]
		pair.LastSubmission = agg.Round
		r.SubmissionPairs[agg.ConfigID] = pair
	}
	log.Debug().Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting done")

	return nil
}

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting")
	proofMap, err := GetProofsAsMap(ctx, aggregates)
	if err != nil || !ValidateAggregateTimestampValues(aggregates) {
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
	hash := chain_utils.Value2HashForSign(aggregate.Value, aggregate.Timestamp.Unix())
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
			return nil, errors.New("proof not found")
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
		return errors.New("unknown message type")
	}
}

func (r *Reporter) reportWithProofs(ctx context.Context, aggregates []GlobalAggregate, proofMap map[int32][]byte) error {
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting with proofs")
	if r.KlaytnHelper == nil {
		return errors.New("klaytn helper not set")
	}

	addresses, values, timestamps, proofs, err := MakeContractArgsWithProofs(aggregates, r.SubmissionPairs, proofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithProofs")
		return err
	}
	log.Debug().Str("Player", "Reporter").Int("proofs", len(proofs)).Msg("contract arguements generated")

	err = r.reportDelegated(ctx, SUBMIT_WITH_PROOFS, addresses, values, timestamps, proofs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, SUBMIT_WITH_PROOFS, addresses, values, timestamps, proofs)
	}
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
	rawTx, err := r.KlaytnHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, functionString, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}
	log.Debug().Str("Player", "Reporter").Str("RawTx", rawTx.String()).Msg("delegated raw tx generated")

	signedTx, err := r.KlaytnHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetSignedFromDelegator")
		return err
	}
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
		return errors.New("failed to report deviation")
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

	log.Debug().Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting deviation done")

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
