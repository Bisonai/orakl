package reporter

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"

	"github.com/rs/zerolog/log"
)

const maxTxSubmissionRetries = 3

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
		DalRestEndpoint:        config.DalRestEndpoint,
		contractAddress:        config.ContractAddress,
		SubmissionInterval:     groupInterval,
		CachedWhitelist:        config.CachedWhitelist,
		deviationThreshold:     deviationThreshold,
		KaiaHelper:             config.KaiaHelper,
		LatestDataMap:          config.LatestDataMap,
		LatestSubmittedDataMap: config.LatestSubmittedDataMap,
		local:                  config.Local,
		stop:                   make(chan struct{}),
	}

	reporter.Pairs = configNames(config.Configs)

	if config.JobType == ReportJob {
		reporter.trackHealth = true
		reporter.Job = func() error {
			return reporter.regularReporterJob(ctx)
		}
	} else {
		reporter.Job = func() error {
			return reporter.deviationJob(ctx)
		}
	}

	return reporter, nil
}

func (r *Reporter) Run(ctx context.Context) {
	log.Info().Msgf("Reporter ticker starting with interval: %v", r.SubmissionInterval)
	ticker := time.NewTicker(r.SubmissionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("Player", "Reporter").Msg("context done, stopping reporter")
			return
		case <-r.stop:
			log.Info().Str("Player", "Reporter").Msgf("reporter with interval %v stopped", r.SubmissionInterval)
			return
		case <-ticker.C:
			go func() {
				err := r.Job()
				if err != nil {
					log.Error().Str("Player", "Reporter").Err(err).Msg("ReporterJob")
				}
			}()
		}
	}
}

// Stop ends Run after its current tick; a job already running finishes (its
// submission is never cancelled mid-flight).
func (r *Reporter) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
	if r.local != nil {
		r.local.health.forget(r)
	}
}

// tracksHealth reports whether this reporter's jobs feed the node's health:
// an in-node regular reporter not stopped (Stop forgot its state).
func (r *Reporter) tracksHealth() bool {
	if r.local == nil || !r.trackHealth {
		return false
	}
	select {
	case <-r.stop:
		return false
	default:
		return true
	}
}

// GetPairs returns a snapshot of the pairs; a job reads it once at its start.
func (r *Reporter) GetPairs() []string {
	r.pairsMu.RLock()
	defer r.pairsMu.RUnlock()
	return append([]string(nil), r.Pairs...)
}

// SetPairs replaces the pairs from the next job on.
func (r *Reporter) SetPairs(pairs []string) {
	r.pairsMu.Lock()
	defer r.pairsMu.Unlock()
	r.Pairs = append([]string(nil), pairs...)
}

func (r *Reporter) regularReporterJob(ctx context.Context) error {
	pairs := r.GetPairs()
	if len(pairs) == 0 {
		return nil
	}
	var pairsMap map[string]SubmissionData
	if r.local != nil {
		var led, lost int
		pairsMap, led, lost = r.local.prepareLed(ctx, r.local.latest(r.LatestDataMap, pairs))
		// only lost pairs: health is left as it was, so a node that resigned
		// for failing submissions doesn't turn healthy (and lead again) at
		// once; it clears once a later round or LocalMaxAge drops them
		if r.tracksHealth() && (led > 0 || lost == 0) {
			r.local.health.prepared(r, r.local.now(), led, len(pairsMap))
		}
		if len(pairsMap) == 0 {
			return nil
		}
	} else {
		var err error
		pairsMap, err = GetLatestDataRest(ctx, r.DalRestEndpoint, pairs)
		if err != nil {
			return err
		}
	}

	err := r.report(ctx, pairsMap)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reporter) deviationJob(ctx context.Context) error {
	r.pruneSubmitted()
	deviatingAggregates := GetDeviatingAggregates(r.LatestSubmittedDataMap, r.LatestDataMap, r.deviationThreshold)
	if r.local != nil {
		deviatingAggregates = r.local.prepare(ctx, deviatingAggregates)
	}
	if len(deviatingAggregates) == 0 {
		return nil
	}
	log.Debug().Str("Player", "Reporter").Msgf("deviating aggregates found: %v", deviatingAggregates)

	err := r.report(ctx, deviatingAggregates)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reporter) report(ctx context.Context, pairs map[string]SubmissionData) error {
	var feedHashes [][32]byte
	var values []*big.Int
	var timestamps []*big.Int
	var proofs [][]byte
	var submittedPairs []string

	for pair, submissionData := range pairs {
		feedHashes = append(feedHashes, submissionData.FeedHash)
		values = append(values, big.NewInt(submissionData.Value))
		timestamps = append(timestamps, big.NewInt(submissionData.AggregateTime))
		proofs = append(proofs, submissionData.Proof)
		submittedPairs = append(submittedPairs, pair)
	}

	dataLen := len(feedHashes)
	wg := sync.WaitGroup{}

	// a report counts as delivered if any batch got through: one feed's bad
	// batch must not make the whole reporter unhealthy
	batches := (dataLen + MAX_REPORT_BATCH_SIZE - 1) / MAX_REPORT_BATCH_SIZE
	failed := 0
	if r.tracksHealth() {
		id := r.local.health.begin(r, r.local.now())
		defer func() { r.local.health.end(r, id, r.local.now(), failed < batches || batches == 0) }()
	}

	// one slot per batch: wg.Wait runs before the channel is read
	errorsChan := make(chan error, batches)
	for start := 0; start < dataLen; start += MAX_REPORT_BATCH_SIZE {
		end := min(start+MAX_REPORT_BATCH_SIZE, dataLen)

		batchFeedHashes := feedHashes[start:end]
		batchValues := values[start:end]
		batchTimestamps := timestamps[start:end]
		batchProofs := proofs[start:end]
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := r.KaiaHelper.SubmitDelegatedFallbackDirect(ctx, r.contractAddress, SUBMIT_WITH_PROOFS, batchFeedHashes, batchValues, batchTimestamps, batchProofs)
			if err != nil {
				errorsChan <- err
			}
		}()
	}
	wg.Wait()
	close(errorsChan)

	shouldRefreshNonce := false

	tmp := []error{}
	for err := range errorsChan {
		failed++
		tmp = append(tmp, err)
		if utils.IsNonceError(err) || errors.Is(err, context.DeadlineExceeded) {
			log.Debug().Err(err).Str("Player", "Reporter").Msg("should refresh nonce")
			shouldRefreshNonce = true
		}
	}

	for i, pair := range submittedPairs {
		r.LatestSubmittedDataMap.Store(pair, values[i].Int64())
	}

	if shouldRefreshNonce {
		log.Debug().Str("Player", "Reporter").Msg("refreshing nonce pool")
		return r.KaiaHelper.FlushNoncePool(ctx)
	}

	if len(tmp) > 0 {
		return mergeErrors(tmp)
	}

	log.Debug().Str("Player", "Reporter").Msgf("reporting done for reporter with interval: %v", r.SubmissionInterval)

	return nil
}

// pruneSubmitted drops pairs no longer configured from LatestSubmittedDataMap,
// so the deviation job never submits a removed pair (an in-flight job may
// still record one after the reload cleaned the map).
func (r *Reporter) pruneSubmitted() {
	pairs := r.GetPairs()
	if len(pairs) == 0 {
		return
	}
	active := make(map[string]struct{}, len(pairs))
	for _, p := range pairs {
		active[p] = struct{}{}
	}
	r.LatestSubmittedDataMap.Range(func(key, _ any) bool {
		if pair, ok := key.(string); ok {
			if _, keep := active[pair]; !keep {
				r.LatestSubmittedDataMap.Delete(key)
			}
		}
		return true
	})
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
