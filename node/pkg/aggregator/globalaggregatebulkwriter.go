package aggregator

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

// PostgreSQL SQLSTATE for deadlock_detected — see
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgDeadlockSQLState = "40P01"

// storeProofsRetryAttempts caps the retry loop for transient deadlocks on
// the proofs BulkUpsert. The handoff race in Stop()/Start() should
// eliminate the common deadlock window already; this is defense in depth
// for any remaining edge cases (concurrent ad-hoc admin writers, future
// migrations running mid-flight, etc.).
const storeProofsRetryAttempts = 3

/*
bulk insert proofs and aggregates into pgsql
*/

type GlobalAggregateBulkWriter struct {
	ReceiveChannels map[string]chan SubmissionData
	Buffer          chan SubmissionData

	LatestDataUpdateInterval time.Duration
	PgsqlBulkInsertInterval  time.Duration

	ctx        context.Context
	cancelFunc context.CancelFunc

	// bulkInsertMu serializes bulkInsert calls so that a slow upsert
	// doesn't allow the next tick to start a second concurrent run.
	// Concurrent BulkUpserts on overlapping (config_id, round) rows
	// were the residual source of 40P01 deadlocks even after the rows
	// were pre-sorted (the ON CONFLICT path acquires locks beyond the
	// caller-controlled order).
	bulkInsertMu sync.Mutex
}

const DefaultPgsqlBulkInsertInterval = 1 * time.Second
const DefaultBufferSize = 2000

type GlobalAggregateBulkWriterConfig struct {
	PgsqlBulkInsertInterval time.Duration
	BufferSize              int
	ConfigNames             []string
}

type GlobalAggregateBulkWriterOption func(*GlobalAggregateBulkWriterConfig)

func WithPgsqlBulkInsertInterval(interval time.Duration) GlobalAggregateBulkWriterOption {
	return func(config *GlobalAggregateBulkWriterConfig) {
		config.PgsqlBulkInsertInterval = interval
	}
}

func WithBufferSize(size int) GlobalAggregateBulkWriterOption {
	return func(config *GlobalAggregateBulkWriterConfig) {
		config.BufferSize = size
	}
}

func WithConfigNames(configNames []string) GlobalAggregateBulkWriterOption {
	return func(config *GlobalAggregateBulkWriterConfig) {
		config.ConfigNames = configNames
	}
}

func NewGlobalAggregateBulkWriter(opts ...GlobalAggregateBulkWriterOption) *GlobalAggregateBulkWriter {
	config := &GlobalAggregateBulkWriterConfig{
		PgsqlBulkInsertInterval: DefaultPgsqlBulkInsertInterval,
		BufferSize:              DefaultBufferSize,
	}
	for _, opt := range opts {
		opt(config)
	}

	result := &GlobalAggregateBulkWriter{
		ReceiveChannels: make(map[string]chan SubmissionData, len(config.ConfigNames)),
		Buffer:          make(chan SubmissionData, config.BufferSize),

		PgsqlBulkInsertInterval: config.PgsqlBulkInsertInterval,
	}

	for _, configName := range config.ConfigNames {
		result.ReceiveChannels[configName] = make(chan SubmissionData)
	}

	return result
}

func (s *GlobalAggregateBulkWriter) Start(ctx context.Context) {
	if s.ctx != nil {
		log.Debug().Str("Player", "Aggregator").Msg("GlobalAggregateBulkWriter already running")
		return
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.ctx = ctxWithCancel

	s.receive(ctxWithCancel)
	s.bulkInsertJob(ctxWithCancel)
}

func (s *GlobalAggregateBulkWriter) Stop() {
	if s.ctx == nil {
		log.Debug().Str("Player", "Aggregator").Msg("GlobalAggregateBulkWriter not running")
		return
	}

	s.cancelFunc()
	s.cancelFunc = nil
	s.ctx = nil

	// Wait for any in-flight bulkInsert to finish before returning. The
	// REFRESH_AGGREGATOR_APP handler calls Stop() then immediately creates
	// a new GlobalAggregateBulkWriter (with its own mutex) and starts it.
	// Without this drain the old goroutine's BulkUpsert and the new one's
	// next tick can both run concurrently against the proofs table — each
	// holds its own per-instance mutex, so they don't serialize, and the
	// ON CONFLICT (config_id, round) DO UPDATE path then deadlocks
	// (SQLSTATE 40P01) when their batches share rows. Acquiring and
	// releasing the mutex here ensures the in-flight query returns first,
	// so the handoff to the next instance is sequential.
	s.bulkInsertMu.Lock()
	s.bulkInsertMu.Unlock()
}

func (s *GlobalAggregateBulkWriter) receive(ctx context.Context) {
	for name := range s.ReceiveChannels {
		go s.receiveEach(ctx, name)
	}
}

func (s *GlobalAggregateBulkWriter) receiveEach(ctx context.Context, configName string) {
	err := db.Subscribe(ctx, keys.SubmissionDataStreamKey(configName), s.ReceiveChannels[configName])
	if err != nil {
		log.Error().Err(err).Str("Player", "Aggregator").Msg("failed to subscribe to submission stream")
	}
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-s.ReceiveChannels[configName]:
			s.Buffer <- data
		}
	}
}

func (s *GlobalAggregateBulkWriter) bulkInsertJob(ctx context.Context) {
	ticker := time.NewTicker(s.PgsqlBulkInsertInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				// Skip this tick if a previous bulkInsert is still
				// running. The buffered Buffer chan will simply hold
				// more items until the next tick drains them, which
				// is the same behavior as before but without the
				// concurrent-writer deadlock window.
				if !s.bulkInsertMu.TryLock() {
					continue
				}
				go func() {
					defer s.bulkInsertMu.Unlock()
					s.bulkInsert(ctx)
				}()
			}
		}
	}()
}

func (s *GlobalAggregateBulkWriter) bulkInsert(ctx context.Context) {
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
	default:
		return
	}
}

func storeProofs(ctx context.Context, proofs []Proof) error {
	if len(proofs) == 0 {
		return nil
	}

	seen := make(map[[2]int32]struct{})
	dedupRows := make([][]any, 0, len(proofs))
	for _, proof := range proofs {
		key := [2]int32{proof.ConfigID, proof.Round}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		dedupRows = append(dedupRows, []any{proof.ConfigID, proof.Round, proof.Proof})
	}

	// Sort by (config_id, round) so concurrent BulkUpsert calls acquire row
	// locks in the same order, avoiding 40P01 deadlocks on the ON CONFLICT
	// path when batches share rows (e.g. P2P-replayed proofs).
	sort.Slice(dedupRows, func(i, j int) bool {
		ai, aj := dedupRows[i][0].(int32), dedupRows[j][0].(int32)
		if ai != aj {
			return ai < aj
		}
		return dedupRows[i][1].(int32) < dedupRows[j][1].(int32)
	})

	// Retry on deadlock (SQLSTATE 40P01). Per PostgreSQL docs, "applications
	// using transactions should be prepared to retry on serialization /
	// deadlock failures." The handoff drain in Stop() removes the common
	// source of deadlocks here, so this loop is defense in depth — at the
	// observed pre-fix rate (~2–4/day) a single retry attempt is enough.
	var lastErr error
	for attempt := 1; attempt <= storeProofsRetryAttempts; attempt++ {
		err := db.BulkUpsert(ctx, "proofs", []string{"config_id", "round", "proof"}, dedupRows, []string{"config_id", "round"}, []string{"proof"})
		if err == nil {
			return nil
		}
		lastErr = err
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != pgDeadlockSQLState {
			return err
		}
		log.Warn().Err(err).Int("attempt", attempt).Msg("proofs BulkUpsert deadlock — retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 50 * time.Millisecond):
		}
	}
	return lastErr
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
