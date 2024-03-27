package reporter

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"

	"github.com/klaytn/klaytn/common"
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

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		return nil, errors.New("SUBMISSION_PROXY_CONTRACT not set")
	}

	reporter := &ReporterNode{
		Raft:            raft,
		contractAddress: contractAddress,
	}
	err = reporter.loadSubmissionPairs(ctx)
	if err != nil {
		return nil, err
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
			log.Error().Str("Player", "Reporter").Err(err).Msg("failed to generate jitter for retry timeout")
			n = big.NewInt(0)
		}
		failureTimeout += failureTimeout + time.Duration(n.Int64())*time.Millisecond
		if failureTimeout > MAX_RETRY_DELAY {
			failureTimeout = MAX_RETRY_DELAY
		}

		err = job()
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("job failed")
			time.Sleep(failureTimeout)
			continue
		}
		return nil
	}
	return errors.New("job failed")
}

func (r *ReporterNode) leaderJob() error {
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
			log.Error().Str("Player", "Reporter").Msg("no valid aggregates to report")
			return nil
		}

		err = r.report(ctx, validAggregates)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("Report")
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

func (r *ReporterNode) resignLeader() {
	r.Raft.StopHeartbeatTicker()
	r.Raft.UpdateRole(raft.Follower)
}

func (r *ReporterNode) handleCustomMessage(msg raft.Message) error {
	// TODO: implement message handling related to validation
	return errors.New("unknown message type")
}

func (r *ReporterNode) getLatestGlobalAggregates(ctx context.Context) ([]GlobalAggregate, error) {
	result, err := r.getLatestGlobalAggregatesRdb(ctx)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("getLatestGlobalAggregatesRdb failed, trying to get from pgsql")
		return r.getLatestGlobalAggregatesPgsql(ctx)
	}
	return result, nil
}

func (r *ReporterNode) getLatestGlobalAggregatesPgsql(ctx context.Context) ([]GlobalAggregate, error) {
	names := make([]string, 0, len(r.SubmissionPairs))
	for name := range r.SubmissionPairs {
		names = append(names, name)
	}

	q := makeGetLatestGlobalAggregatesQuery(names)
	return db.QueryRows[GlobalAggregate](ctx, q, nil)
}

func (r *ReporterNode) getLatestGlobalAggregatesRdb(ctx context.Context) ([]GlobalAggregate, error) {
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
		aggregate.Name = strings.TrimPrefix(keys[i], "globalAggregate:")
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}

func (r *ReporterNode) report(ctx context.Context, aggregates []GlobalAggregate) error {
	if r.KlaytnHelper == nil {
		return errors.New("klaytn helper not set")
	}

	addresses, values, err := r.makeContractArgs(aggregates)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("makeContractArgs")
		return err
	}

	err = r.reportDelegated(ctx, addresses, values)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("reporting directly")
		return r.reportDirect(ctx, addresses, values)
	}
	return nil
}

func (r *ReporterNode) reportDirect(ctx context.Context, args ...interface{}) error {
	rawTx, err := r.KlaytnHelper.MakeDirectTx(ctx, r.contractAddress, FUNCTION_STRING, args...)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("MakeDirectTx")
		return err
	}

	return r.KlaytnHelper.SubmitRawTx(ctx, rawTx)
}

func (r *ReporterNode) reportDelegated(ctx context.Context, args ...interface{}) error {
	rawTx, err := r.KlaytnHelper.MakeFeeDelegatedTx(ctx, r.contractAddress, FUNCTION_STRING, args...)
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
	lastSubmission := r.SubmissionPairs[aggregate.Name].LastSubmission
	if lastSubmission == 0 {
		return true
	}
	return aggregate.Round > lastSubmission
}

func (r *ReporterNode) makeContractArgs(aggregates []GlobalAggregate) ([]common.Address, []*big.Int, error) {
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

func (r *ReporterNode) loadSubmissionPairs(ctx context.Context) error {
	if r.SubmissionPairs == nil {
		r.SubmissionPairs = make(map[string]SubmissionPair)
	}

	submissionAddresses, err := db.QueryRows[SubmissionAddress](ctx, "SELECT * FROM submission_addresses;", nil)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to load submission addresses")
		return err
	}

	if len(submissionAddresses) == 0 {
		log.Error().Str("Player", "Reporter").Msg("no submission addresses found")
		return errors.New("no submission addresses found")
	}

	for _, sa := range submissionAddresses {
		r.SubmissionPairs[sa.Name] = SubmissionPair{LastSubmission: 0, Address: common.HexToAddress(sa.Address)}
	}
	return nil
}

func (r *ReporterNode) SetKlaytnHelper(ctx context.Context) error {
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
