package por

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	chainUtils "bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/fetcher"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

const (
	maxTxSubmissionRetries = 3
	defaultInterval        = 60 * time.Second
	submissionBuffer       = 5 * time.Second
	adapterBaseUrl         = "https://config.orakl.network/adapter"
	aggregatorBaseUrl      = "https://config.orakl.network/aggregator"
)

var urls = map[string]urlEntry{
	"peg-por": {
		"/{CHAIN}/peg-{CHAIN}.por.json",
		"/{CHAIN}/peg.por.json",
	},
	"gp": {
		"/{CHAIN}/gp-{CHAIN}.json",
		"/{CHAIN}/gp.json",
	},
}

func New(ctx context.Context) (*app, error) {
	chain := os.Getenv("POR_CHAIN")
	if chain == "" {
		chain = "baobab"
	}

	providerUrl := os.Getenv("POR_PROVIDER_URL")
	if providerUrl == "" {
		providerUrl = os.Getenv("KAIA_PROVIDER_URL")
		if providerUrl == "" {
			return nil, errorSentinel.ErrPorProviderUrlNotFound
		}
	}

	entries := map[string]entry{}
	for n, u := range urls {
		adapterUrl := adapterBaseUrl + strings.ReplaceAll(u.adapterEndpoint, "{CHAIN}", chain)
		aggregatorUrl := aggregatorBaseUrl + strings.ReplaceAll(u.aggregatorEndpoint, "{CHAIN}", chain)

		ad, err := request.Request[adaptor](request.WithEndpoint(adapterUrl))
		if err != nil {
			return nil, err
		}

		if len(ad.Feeds) == 0 {
			return nil, fmt.Errorf("feeds not found for %s", adapterUrl)
		}

		if ad.Decimals == 0 {
			return nil, fmt.Errorf("decimals not found for %s", adapterUrl)
		}

		ag, err := request.Request[aggregator](request.WithEndpoint(aggregatorUrl))
		if err != nil {
			return nil, err
		}

		d := new(fetcher.Definition)
		err = json.Unmarshal(ad.Feeds[0].Definition, &d)
		if err != nil {
			return nil, err
		}

		e := entry{
			definition: d,
			adapter:    ad,
			aggregator: ag,
		}

		entries[n] = e
	}

	porReporterPk := secrets.GetSecret("POR_REPORTER_PK")
	if porReporterPk == "" {
		return nil, errorSentinel.ErrPorReporterPkNotFound
	}

	chainHelper, err := helper.NewChainHelper(
		ctx,
		helper.WithBlockchainType(helper.Kaia),
		helper.WithReporterPk(porReporterPk),
		helper.WithProviderUrl(providerUrl),
	)
	if err != nil {
		return nil, err
	}

	return &app{
		entries:    entries,
		kaiaHelper: chainHelper,
	}, nil
}

func (a *app) Run(ctx context.Context) {
	go func() {
		port := os.Getenv("POR_PORT")
		if port == "" {
			port = "3000"
		}

		http.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
			// Respond with a simple string
			_, err := w.Write([]byte("Miko POR"))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
		})

		http.HandleFunc("/api/v1/address", func(w http.ResponseWriter, r *http.Request) {
			porReporterPk := secrets.GetSecret("POR_REPORTER_PK")
			addr, err := chainUtils.StringPkToAddressHex(porReporterPk)
			if err != nil {
				log.Error().Err(err).Msg("failed to convert pk to address")
			}
			_, err = w.Write([]byte(addr))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
		})

		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal().Err(err).Msg("failed to start http server")
		}
	}()

	// start jobs
	wg := sync.WaitGroup{}
	for j, e := range a.entries {
		wg.Add(1)
		go func(j string, e entry) {
			defer wg.Done()
			a.startJob(ctx, e)
		}(j, e)
	}
	wg.Wait()
}

func (a *app) startJob(ctx context.Context, entry entry) {
	interval := defaultInterval
	if entry.adapter.Interval != nil {
		interval = time.Duration(*entry.adapter.Interval) * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := retrier.Retry(
				func() error {
					return a.execute(ctx, entry)
				},
				maxRetry,
				initialFailureTimeout,
				maxRetryDelay,
			)
			if err != nil {
				log.Error().Err(err).Msg("error in execute")
			}
		}
	}
}

func (a *app) execute(ctx context.Context, e entry) error {
	v, err := fetcher.FetchSingle(ctx, e.definition)
	if err != nil {
		return err
	}

	lastInfo, err := a.getLastInfo(ctx, e)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch last info")
	}

	now := time.Now()
	if a.shouldReport(e, lastInfo, v, now) {
		roundId, err := a.getRoundId(ctx, e)
		if err != nil {
			return err
		}

		err = a.report(ctx, e, v, roundId)
		if err != nil {
			return err
		}
	} else {
		log.Debug().Msg("no need to report")
	}

	return nil
}

func (a *app) report(ctx context.Context, e entry, submissionValue float64, latestRoundId uint32) error {
	tmp := new(big.Float).SetFloat64(submissionValue)
	submissionValueParam := new(big.Int)
	tmp.Int(submissionValueParam)

	latestRoundIdParam := new(big.Int).SetUint64(uint64(latestRoundId))

	return retrier.Retry(func() error {
		tx, err := a.kaiaHelper.MakeDirectTx(ctx, e.aggregator.Address, submitInterface, latestRoundIdParam, submissionValueParam)
		if err != nil {
			return err
		}
		return a.kaiaHelper.Submit(ctx, tx)
	}, maxRetry, initialFailureTimeout, maxRetryDelay)
}

func (a *app) getLastInfo(ctx context.Context, e entry) (lastInfo, error) {
	rawResult, err := a.kaiaHelper.ReadContract(ctx, e.aggregator.Address, latestRoundDataInterface)
	if err != nil {
		return lastInfo{}, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return lastInfo{}, errorSentinel.ErrPorRawResultCastFail
	}

	updatedAt, ok := rawResultSlice[3].(*big.Int)
	if !ok {
		return lastInfo{}, errorSentinel.ErrPorUpdatedAtCastFail
	}

	answer, ok := rawResultSlice[1].(*big.Int)
	if !ok {
		return lastInfo{}, errorSentinel.ErrPorAnswerCastFail
	}

	return lastInfo{
		UpdatedAt: updatedAt,
		Answer:    answer,
	}, nil
}

func (a *app) shouldReport(e entry, lastInfo lastInfo, newVal float64, fetchedTime time.Time) bool {
	if lastInfo.UpdatedAt == nil || lastInfo.Answer == nil || lastInfo.UpdatedAt.Sign() == 0 && lastInfo.Answer.Sign() == 0 {
		return true
	}

	int64UpdatedAt := lastInfo.UpdatedAt.Int64()
	lastSubmissionTime := time.Unix(int64UpdatedAt, 0)
	log.Debug().Msg("time since last submission: " + fetchedTime.Sub(lastSubmissionTime).String())

	submitInterval := 60 * time.Minute
	if e.aggregator.Heartbeat != nil {
		submitInterval = time.Duration(*e.aggregator.Heartbeat) * time.Millisecond
	}

	if fetchedTime.Sub(lastSubmissionTime) > submitInterval-submissionBuffer {
		return true
	}

	oldVal, _ := new(big.Float).SetInt(lastInfo.Answer).Float64()

	log.Debug().Float64("oldValue", oldVal).Float64("newValue", newVal).Msg("checking deviation")

	denominator := math.Pow10(e.adapter.Decimals)
	if denominator == 0 {
		log.Error().Float64("denom", denominator).Msg("invalid denominator")
		return false
	}

	o, n := oldVal/denominator, newVal/denominator

	if o != 0 && n != 0 {
		deviationRange := oldVal * e.aggregator.Threshold
		min := oldVal - deviationRange
		max := oldVal + deviationRange
		return newVal < min || newVal > max
	} else if o == 0 && n != 0 {
		return newVal > e.aggregator.AbsoluteThreshold
	} else {
		return false
	}
}

func (a *app) getRoundId(ctx context.Context, e entry) (uint32, error) {
	publicAddress, err := a.kaiaHelper.PublicAddress()
	if err != nil {
		return 0, err
	}

	rawResult, err := a.kaiaHelper.ReadContract(ctx, e.aggregator.Address, oracleRoundStateInterface, publicAddress, uint32(0))
	if err != nil {
		return 0, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return 0, errorSentinel.ErrPorRawResultCastFail
	}

	RoundID, ok := rawResultSlice[1].(uint32)
	if !ok {
		return 0, errorSentinel.ErrPorRoundIdCastFail
	}

	return RoundID, nil
}
