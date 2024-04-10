package reporter

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"

	"github.com/klaytn/klaytn/common"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewReporter(ctx context.Context, h host.Host, ps *pubsub.PubSub, submissionPairs []SubmissionAddress, interval int) (*Reporter, error) {
	if len(submissionPairs) == 0 {
		log.Error().Str("Player", "Reporter").Err(errors.New("no submission pairs")).Msg("no submission pairs to make new reporter")
		return nil, errors.New("no submission pairs")
	}
	topicString := TOPIC_STRING + "-" + strconv.Itoa(interval)
	groupInterval := time.Duration(interval) * time.Millisecond
	topic, err := ps.Join(topicString)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	raft := raft.NewRaftNode(h, ps, topic, MESSAGE_BUFFER, groupInterval)

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return nil, errors.New("SUBMISSION_PROXY_CONTRACT not set")
	}

	reporter := &Reporter{
		Raft:               raft,
		contractAddress:    contractAddress,
		SubmissionInterval: groupInterval,
	}

	reporter.SubmissionPairs = make(map[string]SubmissionPair)
	for _, sa := range submissionPairs {
		reporter.SubmissionPairs[sa.Name] = SubmissionPair{LastSubmission: 0, Address: common.HexToAddress(sa.Address)}
	}

	reporter.Raft.LeaderJob = reporter.leaderJob
	reporter.Raft.HandleCustomMessage = reporter.handleCustomMessage

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	r.Raft.Run(ctx)
}

func (r *Reporter) retry(job func() error) error {
	failureTimeout := INITIAL_FAILURE_TIMEOUT
	for i := 0; i < MAX_RETRY; i++ {

		failureTimeout = calculateJitter(failureTimeout)
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

	job := func() error {
		aggregates, err := r.getLatestGlobalAggregates(ctx)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("GetLatestGlobalAggregates")
			return err
		}

		validAggregates := r.filterInvalidAggregates(aggregates)
		if len(validAggregates) == 0 {
			log.Warn().Str("Player", "Reporter").Msg("no valid aggregates to report")
			return nil
		}
		log.Debug().Str("Player", "Reporter").Int("validAggregates", len(validAggregates)).Msg("valid aggregates")

		err = r.report(ctx, validAggregates)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("report")
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

	err := r.retry(job)
	if err != nil {
		r.resignLeader()
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to report")
		return errors.New("failed to report")
	}

	return nil
}

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {
	proofMap, err := r.getProofsAsMap(ctx, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return r.reportWithoutProofs(ctx, aggregates)
	}

	return r.reportWithProofs(ctx, aggregates, proofMap)
}

func (r *Reporter) getProofsAsMap(ctx context.Context, aggregates []GlobalAggregate) (map[string][]byte, error) {
	proofs, err := r.getProofs(ctx, aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("submit without proofs")
		return nil, err
	}

	if len(proofs) < len(aggregates) {
		log.Error().Str("Player", "Reporter").Msg("proofs not found for all aggregates")
		return nil, errors.New("proofs not found for all aggregates")
	}

	return ProofsToMap(proofs), nil
}

func (r *Reporter) resignLeader() {
	r.Raft.StopHeartbeatTicker()
	r.Raft.UpdateRole(raft.Follower)
}

func (r *Reporter) handleCustomMessage(msg raft.Message) error {
	// TODO: implement message handling related to validation
	return errors.New("unknown message type")
}

func (r *Reporter) getLatestGlobalAggregates(ctx context.Context) ([]GlobalAggregate, error) {
	result, err := r.getLatestGlobalAggregatesRdb(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregatesRdb failed, trying to get from pgsql")
		return r.getLatestGlobalAggregatesPgsql(ctx)
	}
	return result, nil
}

func (r *Reporter) getLatestGlobalAggregatesPgsql(ctx context.Context) ([]GlobalAggregate, error) {
	names := make([]string, 0, len(r.SubmissionPairs))
	for name := range r.SubmissionPairs {
		names = append(names, name)
	}

	q := makeGetLatestGlobalAggregatesQuery(names)
	return db.QueryRows[GlobalAggregate](ctx, q, nil)
}

func (r *Reporter) getLatestGlobalAggregatesRdb(ctx context.Context) ([]GlobalAggregate, error) {
	keys := make([]string, 0, len(r.SubmissionPairs))

	for name := range r.SubmissionPairs {
		keys = append(keys, "globalAggregate:"+name)
	}

	result, err := db.MGet(ctx, keys)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get latest global aggregates")
		return nil, err
	}

	aggregates := make([]GlobalAggregate, 0, len(result))
	for i, agg := range result {
		if agg == nil {
			log.Error().Str("Player", "Reporter").Str("key", keys[i]).Msg("missing aggregate")
			continue
		}
		var aggregate GlobalAggregate
		err = json.Unmarshal([]byte(agg.(string)), &aggregate)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Str("key", keys[i]).Msg("failed to unmarshal aggregate")
			continue
		}
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}

func (r *Reporter) getProofs(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	result, err := r.getProofsRdb(ctx, aggregates)
	if err != nil {
		log.Warn().Str("Player", "Reporter").Err(err).Msg("getProofsRdb failed, trying to get from pgsql")
		return r.getProofsPgsql(ctx, aggregates)
	}
	return result, nil
}

func (r *Reporter) getProofsRdb(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	keys := make([]string, 0, len(aggregates))
	for _, agg := range aggregates {
		keys = append(keys, "proof:"+agg.Name+"|round:"+strconv.FormatInt(agg.Round, 10))
	}

	result, err := db.MGet(ctx, keys)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get proofs")
		return nil, err
	}

	proofs := make([]Proof, 0, len(result))
	for i, proof := range result {
		if proof == nil {
			log.Error().Str("Player", "Reporter").Str("key", keys[i]).Msg("missing proof")
			continue
		}
		var p Proof
		err = json.Unmarshal([]byte(proof.(string)), &p)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Str("key", keys[i]).Msg("failed to unmarshal proof")
			continue
		}
		proofs = append(proofs, p)

	}
	return proofs, nil
}

func (r *Reporter) getProofsPgsql(ctx context.Context, aggregates []GlobalAggregate) ([]Proof, error) {
	q := makeGetProofsQuery(aggregates)
	rawResult, err := db.QueryRows[PgsqlProof](ctx, q, nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to get proofs")
		return nil, err
	}
	return convertPgsqlProofsToProofs(rawResult), nil
}

func (r *Reporter) reportWithoutProofs(ctx context.Context, aggregates []GlobalAggregate) error {
	log.Debug().Str("Player", "Reporter").Int("aggregates", len(aggregates)).Msg("reporting without proofs")
	if r.KlaytnHelper == nil {
		log.Error().Str("Player", "Reporter").Msg("klaytn helper not set")
		return errors.New("klaytn helper not set")
	}

	addresses, values, err := r.makeContractArgsWithoutProofs(aggregates)
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

	addresses, values, proofs, err := r.makeContractArgsWithProofs(aggregates, proofMap)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgsWithProofs")
		return err
	}

	err = r.reportDelegated(ctx, SUBMIT_WITH_PROOFS, addresses, values, proofs)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, SUBMIT_WITH_PROOFS, addresses, values, proofs)
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

func (r *Reporter) filterInvalidAggregates(aggregates []GlobalAggregate) []GlobalAggregate {
	validAggregates := make([]GlobalAggregate, 0, len(aggregates))
	for _, agg := range aggregates {
		if r.isAggValid(agg) {
			validAggregates = append(validAggregates, agg)
		}
	}
	return validAggregates
}

func (r *Reporter) isAggValid(aggregate GlobalAggregate) bool {
	lastSubmission := r.SubmissionPairs[aggregate.Name].LastSubmission
	if lastSubmission == 0 {
		return true
	}
	return aggregate.Round > lastSubmission
}

func (r *Reporter) makeContractArgsWithoutProofs(aggregates []GlobalAggregate) ([]common.Address, []*big.Int, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	for i, agg := range aggregates {
		if agg.Name == "" || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Str("name", agg.Name).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = r.SubmissionPairs[agg.Name].Address
		values[i] = big.NewInt(agg.Value)
	}

	if len(addresses) == 0 || len(values) == 0 {
		return nil, nil, errors.New("no valid aggregates")
	}

	return addresses, values, nil
}

func (r *Reporter) makeContractArgsWithProofs(aggregates []GlobalAggregate, proofMap map[string][]byte) ([]common.Address, []*big.Int, [][]byte, error) {
	addresses := make([]common.Address, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	proofs := make([][]byte, len(aggregates))

	for i, agg := range aggregates {
		if agg.Name == "" || agg.Value < 0 {
			log.Error().Str("Player", "Reporter").Str("name", agg.Name).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, nil, errors.New("invalid aggregate exists")
		}
		addresses[i] = r.SubmissionPairs[agg.Name].Address
		values[i] = big.NewInt(agg.Value)
		proofs[i] = proofMap[agg.Name+"-"+strconv.FormatInt(agg.Round, 10)]

	}

	if len(addresses) == 0 || len(values) == 0 || len(proofs) == 0 {
		return nil, nil, nil, errors.New("no valid aggregates")
	}

	return addresses, values, proofs, nil
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

func calculateJitter(baseTimeout time.Duration) time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to generate jitter for retry timeout")
		return baseTimeout
	}
	jitter := time.Duration(n.Int64()) * time.Millisecond
	return baseTimeout + jitter
}

func convertPgsqlProofsToProofs(pgsqlProofs []PgsqlProof) []Proof {
	proofs := make([]Proof, len(pgsqlProofs))
	for i, pgsqlProof := range pgsqlProofs {
		proofs[i] = Proof{
			Name:  pgsqlProof.Name,
			Round: pgsqlProof.Round,
			Proof: pgsqlProof.Proof,
		}
	}
	return proofs
}

func concatBytes(slices [][]byte) []byte {
	var result []byte
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

func ProofsToMap(proofs []Proof) map[string][]byte {
	m := make(map[string][]byte)
	for _, proof := range proofs {
		//m[name-round] = proof
		m[proof.Name+"-"+strconv.FormatInt(proof.Round, 10)] = proof.Proof
	}
	return m
}
