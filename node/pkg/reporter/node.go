package reporter

import (
	"context"
	"errors"
	"math/big"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const (
	TOPIC_STRING    = "orakl-offchain-aggregation-reporter"
	LEADER_TIMEOUT  = 5 * time.Second
	MAX_RETRY       = 3
	FUNCTION_STRING = "batchSubmit(string[] memory _pairs, int256[] memory _prices)"

	GET_LATEST_GLOBAL_AGGREGATES_QUERY = `
		SELECT ga.name, ga.value, ga.round, ga.timestamp
		FROM global_aggregates ga
		JOIN (
			SELECT name, MAX(round) as max_round
			FROM global_aggregates
			GROUP BY name
		) subq ON ga.name = subq.name AND ga.round = subq.max_round;`
)

type Reporter struct {
	Raft     *raft.Raft
	TxHelper *utils.TxHelper

	lastSubmissions map[string]int64
	contractAddress string
}

type GlobalAggregate struct {
	Name      string    `db:"name"`
	Value     int64     `db:"value"`
	Round     int64     `db:"round"`
	Timestamp time.Time `db:"timestamp"`
}

func New(ctx context.Context, h host.Host, ps *pubsub.PubSub) (*Reporter, error) {
	encryptedTopic, err := utils.EncryptText(TOPIC_STRING)
	if err != nil {
		return nil, err
	}

	topic, err := ps.Join(encryptedTopic)
	if err != nil {
		return nil, err
	}

	raft := raft.NewRaftNode(h, ps, topic, 100, LEADER_TIMEOUT)
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		return nil, err
	}

	reporter := &Reporter{
		Raft:            raft,
		TxHelper:        txHelper,
		contractAddress: os.Getenv("SUBMISSION_PROXY_CONTRACT"),
		lastSubmissions: make(map[string]int64),
	}
	reporter.Raft.LeaderJob = reporter.leaderJob
	reporter.Raft.HandleCustomMessage = reporter.handleCustomMessage

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	r.Raft.Run(ctx)
}

func (r *Reporter) leaderJob() error {
	for i := 0; i < MAX_RETRY; i++ {
		aggregates, err := r.getLatestGlobalAggregates(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("GetLatestGlobalAggregates")
			continue
		}

		validAggregates := r.filterInvalidAggregates(aggregates)
		if len(validAggregates) == 0 {
			log.Error().Msg("no valid aggregates to report")
			continue
		}

		err = r.report(context.Background(), validAggregates)
		if err != nil {
			log.Error().Err(err).Msg("Report")
			continue
		}

		for _, agg := range validAggregates {
			r.lastSubmissions[agg.Name] = agg.Round
		}
		return nil
	}
	r.resignLeader()
	return errors.New("failed to report")
}

func (r *Reporter) resignLeader() {
	r.Raft.StopHeartbeatTicker()
	r.Raft.UpdateRole(raft.Follower)
}

func (r *Reporter) handleCustomMessage(msg raft.Message) error {
	return errors.New("unknown message type")
}

func (r *Reporter) getLatestGlobalAggregates(ctx context.Context) ([]GlobalAggregate, error) {
	return db.QueryRows[GlobalAggregate](ctx, GET_LATEST_GLOBAL_AGGREGATES_QUERY, nil)
}

func (r *Reporter) report(ctx context.Context, aggregates []GlobalAggregate) error {
	pairs, values := r.makeContractArgs(aggregates)
	rawTx, err := r.TxHelper.MakeDirectTx(ctx, r.contractAddress, FUNCTION_STRING, pairs, values)
	if err != nil {
		log.Error().Err(err).Msg("MakeDirectTx")
		return err
	}
	return r.TxHelper.SubmitRawTx(ctx, rawTx)
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
	lastSubmission, ok := r.lastSubmissions[aggregate.Name]
	if !ok {
		return true
	}
	return aggregate.Round > lastSubmission
}

func (r *Reporter) makeContractArgs(aggregates []GlobalAggregate) ([]string, []*big.Int) {
	pairs := make([]string, len(aggregates))
	values := make([]*big.Int, len(aggregates))
	for i, agg := range aggregates {
		pairs[i] = agg.Name
		values[i] = big.NewInt(agg.Value)
	}
	return pairs, values
}
