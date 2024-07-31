package aggregator

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
)

/*
1. bulk insert proofs and aggregates into pgsql
2. update latest proof, aggregate in rdb
*/

type Streamer struct {
	ReceiveChannels map[int32]chan SubmissionData
	Buffer          chan SubmissionData

	LatestDataUpdateInterval time.Duration
	PgsqlBulkInsertInterval  time.Duration

	ctx        context.Context
	cancelFunc context.CancelFunc
}

const DefaultPgsqlBulkInsertInterval = 1 * time.Second
const DefaultBufferSize = 2000

type StreamerConfig struct {
	PgsqlBulkInsertInterval time.Duration
	BufferSize              int
	ConfigIds               []int32
}

type StreamerOption func(*StreamerConfig)

func WithPgsqlBulkInsertInterval(interval time.Duration) StreamerOption {
	return func(config *StreamerConfig) {
		config.PgsqlBulkInsertInterval = interval
	}
}

func WithBufferSize(size int) StreamerOption {
	return func(config *StreamerConfig) {
		config.BufferSize = size
	}
}

func WithConfigIds(configIds []int32) StreamerOption {
	return func(config *StreamerConfig) {
		config.ConfigIds = configIds
	}
}

func NewStreamer(opts ...StreamerOption) *Streamer {
	config := &StreamerConfig{
		PgsqlBulkInsertInterval: DefaultPgsqlBulkInsertInterval,
		BufferSize:              DefaultBufferSize,
	}
	for _, opt := range opts {
		opt(config)
	}

	result := &Streamer{
		ReceiveChannels: make(map[int32]chan SubmissionData, len(config.ConfigIds)),
		Buffer:          make(chan SubmissionData, config.BufferSize),

		PgsqlBulkInsertInterval: config.PgsqlBulkInsertInterval,
	}

	for _, configId := range config.ConfigIds {
		result.ReceiveChannels[configId] = make(chan SubmissionData)
	}

	return result
}

func (s *Streamer) Start(ctx context.Context) {
	if s.ctx != nil {
		log.Debug().Str("Player", "Aggregator").Msg("Streamer already running")
		return
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.ctx = ctxWithCancel

	s.receive(ctxWithCancel)
	s.bulkInsertJob(ctxWithCancel)
}

func (s *Streamer) Stop() {
	if s.ctx == nil {
		log.Debug().Str("Player", "Aggregator").Msg("Streamer not running")
		return
	}

	s.cancelFunc()
	s.cancelFunc = nil
	s.ctx = nil
}

func (s *Streamer) receive(ctx context.Context) {
	for id := range s.ReceiveChannels {
		go s.receiveEach(ctx, id)
	}
}

func (s *Streamer) receiveEach(ctx context.Context, configId int32) {
	err := db.Subscribe(ctx, keys.SubmissionDataStreamKey(configId), s.ReceiveChannels[configId])
	if err != nil {
		log.Error().Err(err).Str("Player", "Aggregator").Msg("failed to subscribe to submission stream")
	}
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-s.ReceiveChannels[configId]:
			s.Buffer <- data
		}
	}
}

func (s *Streamer) bulkInsertJob(ctx context.Context) {
	ticker := time.NewTicker(s.PgsqlBulkInsertInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				go s.bulkInsert(ctx)
			}
		}
	}()
}

func (s *Streamer) bulkInsert(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case submissionData := <-s.Buffer:
		proofBatch := []Proof{submissionData.Proof}
		globalAggregateBatch := []GlobalAggregate{submissionData.GlobalAggregate}
	loop:
		for {
			select {
			case submissionData := <-s.Buffer:
				proofBatch = append(proofBatch, submissionData.Proof)
				globalAggregateBatch = append(globalAggregateBatch, submissionData.GlobalAggregate)
			default:
				break loop
			}
		}
		err := storeProofs(ctx, proofBatch)
		if err != nil {
			log.Error().Err(err).Msg("failed to store proofs")
		}
		err = storeGlobalAggregates(ctx, globalAggregateBatch)
		if err != nil {
			log.Error().Err(err).Msg("failed to store global aggregates")
		}
	}
}

func storeProofs(ctx context.Context, proofs []Proof) error {
	if len(proofs) == 0 {
		return nil
	}

	insertRows := make([][]any, len(proofs))
	for i, proof := range proofs {
		insertRows[i] = []any{proof.ConfigID, proof.Round, proof.Proof}
	}

	_, err := db.BulkCopy(ctx, "proofs", []string{"config_id", "round", "proof"}, insertRows)
	return err
}

func storeGlobalAggregates(ctx context.Context, globalAggregates []GlobalAggregate) error {
	if len(globalAggregates) == 0 {
		return nil
	}

	insertRows := make([][]any, len(globalAggregates))
	for i, globalAggregate := range globalAggregates {
		insertRows[i] = []any{globalAggregate.Value, globalAggregate.Timestamp, globalAggregate.Round, globalAggregate.ConfigID}
	}

	_, err := db.BulkCopy(ctx, "global_aggregates", []string{"value", "timestamp", "round", "config_id"}, insertRows)
	return err
}
