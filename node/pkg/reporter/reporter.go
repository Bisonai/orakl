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

	"github.com/klaytn/klaytn/common"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewReporter(ctx context.Context, h host.Host, ps *pubsub.PubSub, submissionPairs []SubmissionAddress, interval int, contractAddress string, cachedWhitelist []common.Address) (*Reporter, error) {
	topicString := TOPIC_STRING + "-" + strconv.Itoa(interval)
	groupInterval := time.Duration(interval) * time.Millisecond

	reporter, err := newReporter(ctx, h, ps, submissionPairs, groupInterval, topicString, contractAddress, cachedWhitelist)
	if err != nil {
		return nil, err
	}

	reporter.Raft.LeaderJob = reporter.leaderJob
	return reporter, nil
}

func NewDeviationReporter(ctx context.Context, h host.Host, ps *pubsub.PubSub, submissionPairs []SubmissionAddress, contractAddress string, cachedWhitelist []common.Address) (*Reporter, error) {
	topicString := TOPIC_STRING + "-deviation"

	reporter, err := newReporter(ctx, h, ps, submissionPairs, DEVIATION_TIMEOUT, topicString, contractAddress, cachedWhitelist)
	if err != nil {
		return nil, err
	}

	reporter.Raft.LeaderJob = reporter.deviationJob
	return reporter, nil
}

func newReporter(ctx context.Context, h host.Host, ps *pubsub.PubSub, submissionPairs []SubmissionAddress, interval time.Duration, topicString string, contractAddress string, cachedWhitelist []common.Address) (*Reporter, error) {
	if len(submissionPairs) == 0 {
		log.Error().Str("Player", "Reporter").Err(errors.New("no submission pairs")).Msg("no submission pairs to make new reporter")
		return nil, errors.New("no submission pairs")
	}

	topic, err := ps.Join(topicString)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	raft := raft.NewRaftNode(h, ps, topic, MESSAGE_BUFFER, interval)

	reporter := &Reporter{
		Raft:               raft,
		contractAddress:    contractAddress,
		SubmissionInterval: interval,
		CachedWhitelist:    cachedWhitelist,
	}

	reporter.SubmissionPairs = make(map[string]SubmissionPair)
	for _, sa := range submissionPairs {
		reporter.SubmissionPairs[sa.Name] = SubmissionPair{LastSubmission: 0, Address: common.HexToAddress(sa.Address)}
	}
	reporter.Raft.HandleCustomMessage = reporter.handleCustomMessage

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	r.Raft.Run(ctx)
}

func (r *Reporter) retry(job func() error) error {
	failureTimeout := INITIAL_FAILURE_TIMEOUT
	for i := 0; i < MAX_RETRY; i++ {

		failureTimeout = CalculateJitter(failureTimeout)
		if failureTimeout > MAX_RETRY_DELAY {
			failureTimeout = MAX_RETRY_DELAY
		}

		err := job()
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("job failed, retrying")
			time.Sleep(failureTimeout)
			continue
		}
		return nil
	}
	log.Error().Str("Player", "Reporter").Msg("job failed")
	return errors.New("job failed")
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

	err = r.retry(reportJob)
	if err != nil {
		r.resignLeader()
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report")
		return errors.New("failed to report")
	}

	err = r.PublishSubmissionMessage(validAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("PublishSubmissionMessage")
		return err
	}

	for _, agg := range validAggregates {
		pair := r.SubmissionPairs[agg.Name]
		pair.LastSubmission = agg.Round
		r.SubmissionPairs[agg.Name] = pair
	}
	log.Debug().Str("Player", "Reporter").Dur("duration", time.Since(start)).Msg("reporting done")

	return nil
}

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {
	proofMap, err := GetProofsAsMap(ctx, aggregates)
	if err != nil || !ValidateAggregateTimestampValues(aggregates) {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return r.reportWithoutProofs(ctx, aggregates)
	}

	orderedProofMap, err := r.orderProofs(ctx, proofMap, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("orderProofs")
		return r.reportWithoutProofs(ctx, aggregates)
	}

	err = UpdateProofs(ctx, aggregates, orderedProofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("updateProofs")
		return r.reportWithoutProofs(ctx, aggregates)
	}

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

func (r *Reporter) orderProofs(ctx context.Context, proofMap map[string][]byte, aggregates []GlobalAggregate) (map[string][]byte, error) {
	orderedProofMap := make(map[string][]byte)
	for _, agg := range aggregates {
		proof, ok := proofMap[agg.Name]
		if !ok {
			log.Error().Str("Player", "Reporter").Msg("proof not found")
			return nil, errors.New("proof not found")
		}

		orderedProof, err := r.orderProof(ctx, proof, agg)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("orderProof")
			return nil, err
		}

		orderedProofMap[agg.Name] = orderedProof
	}

	return orderedProofMap, nil
}

func (r *Reporter) resignLeader() {
	r.Raft.StopHeartbeatTicker()
	r.Raft.UpdateRole(raft.Follower)
}

func (r *Reporter) handleCustomMessage(ctx context.Context, msg raft.Message) error {
	switch msg.Type {
	case SubmissionMsg:
		return r.HandleSubmissionMessage(ctx, msg)
	default:
		return errors.New("unknown message type")
	}
}

func (r *Reporter) reportWithoutProofs(ctx context.Context, aggregates []GlobalAggregate) error {
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting without proofs")
	if r.KlaytnHelper == nil {
		log.Error().Str("Player", "Reporter").Msg("klaytn helper not set")
		return errors.New("klaytn helper not set")
	}

	addresses, values, err := MakeContractArgsWithoutProofs(aggregates, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithoutProofs")
		return err
	}

	err = r.reportDelegated(ctx, SUBMIT_WITHOUT_PROOFS, addresses, values)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, SUBMIT_WITHOUT_PROOFS, addresses, values)
	}
	return nil
}

func (r *Reporter) reportWithProofs(ctx context.Context, aggregates []GlobalAggregate, proofMap map[string][]byte) error {
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting with proofs")
	if r.KlaytnHelper == nil {
		return errors.New("klaytn helper not set")
	}

	addresses, values, proofs, timestamps, err := MakeContractArgsWithProofs(aggregates, r.SubmissionPairs, proofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithProofs")
		return err
	}

	err = r.reportDelegated(ctx, SUBMIT_WITH_PROOFS, addresses, values, proofs, timestamps)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, SUBMIT_WITH_PROOFS, addresses, values, proofs, timestamps)
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
	rawTx, err := r.KlaytnHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, functionString, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}

	signedTx, err := r.KlaytnHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("GetSignedFromDelegator")
		return err
	}

	return r.KlaytnHelper.SubmitRawTx(ctx, signedTx)
}

func (r *Reporter) SetKlaytnHelper(ctx context.Context) error {
	if r.KlaytnHelper != nil {
		r.KlaytnHelper.Close()
	}
	klaytnHelper, err := helper.NewKlayHelper(ctx, "")
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
	lastAggregates, err := GetLatestGlobalAggregates(ctx, r.SubmissionPairs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregates")
		return err
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

	err = r.retry(reportJob)
	if err != nil {
		r.resignLeader()
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report deviation")
		return errors.New("failed to report deviation")
	}

	err = r.PublishSubmissionMessage(deviatingAggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("PublishSubmissionMessage")
		return err
	}

	for _, agg := range deviatingAggregates {
		pair := r.SubmissionPairs[agg.Name]
		pair.LastSubmission = agg.Round
		r.SubmissionPairs[agg.Name] = pair
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
