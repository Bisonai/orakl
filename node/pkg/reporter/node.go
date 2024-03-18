package reporter

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/klaytn_helper"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewNode(ctx context.Context, h host.Host, ps *pubsub.PubSub) (*ReporterNode, error) {
	topicString := TOPIC_STRING

	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	raft := raft.NewRaftNode(h, ps, topic, MESSAGE_BUFFER, LEADER_TIMEOUT)
	txHelper, err := klaytn_helper.NewTxHelper(ctx)
	if err != nil {
		return nil, err
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return nil, errors.New("SUBMISSION_PROXY_CONTRACT not set")
	}

	reporter := &ReporterNode{
		Raft:            raft,
		TxHelper:        txHelper,
		contractAddress: contractAddress,
		lastSubmissions: make(map[string]int64),
	}
	reporter.Raft.LeaderJob = reporter.leaderJob
	reporter.Raft.HandleCustomMessage = reporter.handleCustomMessage

	return reporter, nil
}

func (r *ReporterNode) Run(ctx context.Context) {
	r.Raft.Run(ctx)
}

func (r *ReporterNode) retry(job func() error) error {
	failureTimeout := INITIAL_FAILURE_TIMEOUT
	for i := 0; i < MAX_RETRY; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			log.Error().Err(err).Msg("failed to generate jitter for retry timeout")
			n = big.NewInt(0)
		}
		failureTimeout += failureTimeout + time.Duration(n.Int64())*time.Millisecond
		if failureTimeout > MAX_RETRY_DELAY {
			failureTimeout = MAX_RETRY_DELAY
		}

		err = job()
		if err != nil {
			log.Error().Err(err).Msg("job failed")
			time.Sleep(failureTimeout)
			continue
		}
		return nil
	}
	return errors.New("job failed")
}

func (r *ReporterNode) leaderJob() error {
	ctx := context.Background()

	job := func() error {
		aggregates, err := r.getLatestGlobalAggregates(ctx)
		if err != nil {
			log.Error().Err(err).Msg("GetLatestGlobalAggregates")
			return err
		}

		validAggregates := r.filterInvalidAggregates(aggregates)
		if len(validAggregates) == 0 {
			log.Error().Msg("no valid aggregates to report")
			return nil
		}

		err = r.report(ctx, validAggregates)
		if err != nil {
			log.Error().Err(err).Msg("Report")
			return err
		}

		for _, agg := range validAggregates {
			r.lastSubmissions[agg.Name] = agg.Round
		}
		return nil
	}

	err := r.retry(job)
	if err != nil {
		r.resignLeader()
		log.Error().Err(err).Msg("failed to report")
		return errors.New("failed to report")
	}

	return nil
}

func (r *ReporterNode) resignLeader() {
	r.Raft.StopHeartbeatTicker()
	r.Raft.UpdateRole(raft.Follower)
}

func (r *ReporterNode) handleCustomMessage(msg raft.Message) error {
	// TODO: implement message handling related to validation
	return errors.New("unknown message type")
}

func (r *ReporterNode) getLatestGlobalAggregates(ctx context.Context) ([]GlobalAggregate, error) {
	return db.QueryRows[GlobalAggregate](ctx, GET_LATEST_GLOBAL_AGGREGATES_QUERY, nil)
}

func (r *ReporterNode) report(ctx context.Context, aggregates []GlobalAggregate) error {
	pairs, values, err := r.makeContractArgs(aggregates)
	if err != nil {
		log.Error().Err(err).Msg("makeContractArgs")
		return err
	}

	err = r.reportDelegated(ctx, pairs, values)
	if err != nil {
		log.Error().Err(err).Msg("reportDelegated failed")
		return r.reportDirect(ctx, pairs, values)
	}
	return nil
}

func (r *ReporterNode) reportDirect(ctx context.Context, args ...interface{}) error {
	rawTx, err := r.TxHelper.MakeDirectTx(ctx, r.contractAddress, FUNCTION_STRING, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirectTx")
		return err
	}

	return r.TxHelper.SubmitRawTx(ctx, rawTx)
}

func (r *ReporterNode) reportDelegated(ctx context.Context, args ...interface{}) error {
	rawTx, err := r.TxHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, FUNCTION_STRING, args...)
	if err != nil {
		log.Error().Err(err).Msg("MakeFeeDelegatedTx")
		return err
	}

	signedTx, err := r.TxHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		log.Error().Err(err).Msg("GetSignedFromDelegator")
		return err
	}

	return r.TxHelper.SubmitRawTx(ctx, signedTx)
}

func (r *ReporterNode) filterInvalidAggregates(aggregates []GlobalAggregate) []GlobalAggregate {
	validAggregates := make([]GlobalAggregate, 0, len(aggregates))
	for _, agg := range aggregates {
		if r.isAggValid(agg) {
			validAggregates = append(validAggregates, agg)
		}
	}
	return validAggregates
}

func (r *ReporterNode) isAggValid(aggregate GlobalAggregate) bool {
	lastSubmission, ok := r.lastSubmissions[aggregate.Name]
	if !ok {
		return true
	}
	return aggregate.Round > lastSubmission
}

func (r *ReporterNode) makeContractArgs(aggregates []GlobalAggregate) ([]string, []*big.Int, error) {
	pairs := make([]string, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	for i, agg := range aggregates {
		if agg.Name == "" || agg.Value < 0 {
			log.Error().Str("name", agg.Name).Int64("value", agg.Value).Msg("skipping invalid aggregate")
			return nil, nil, errors.New("invalid aggregate exists")
		}
		pairs[i] = agg.Name
		values[i] = big.NewInt(agg.Value)
	}

	if len(pairs) == 0 || len(values) == 0 {
		return nil, nil, errors.New("no valid aggregates")
	}

	return pairs, values, nil
}
