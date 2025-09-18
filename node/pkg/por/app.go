package por

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"bisonai.com/miko/node/pkg/chain/helper"
	chainUtils "bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/db"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/fetcher"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/utils/retrier"
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
		false,
	},
	"gp": {
		"/{CHAIN}/gp-{CHAIN}.json",
		"/{CHAIN}/gp.json",
		false,
	},
	"aapl": {
		"/{CHAIN}/aapl-{CHAIN}.json",
		"/{CHAIN}/aapl.json",
		true,
	},
	"amzn": {
		"/{CHAIN}/amzn-{CHAIN}.json",
		"/{CHAIN}/amzn.json",
		true,
	},
	"googl": {
		"/{CHAIN}/googl-{CHAIN}.json",
		"/{CHAIN}/googl.json",
		true,
	},
	"meta": {
		"/{CHAIN}/meta-{CHAIN}.json",
		"/{CHAIN}/meta.json",
		true,
	},
	"msft": {
		"/{CHAIN}/msft-{CHAIN}.json",
		"/{CHAIN}/msft.json",
		true,
	},
	"nvda": {
		"/{CHAIN}/nvda-{CHAIN}.json",
		"/{CHAIN}/nvda.json",
		true,
	},
	"tsla": {
		"/{CHAIN}/tsla-{CHAIN}.json",
		"/{CHAIN}/tsla.json",
		true,
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
			useProxy:   u.useProxy,
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

	proxies, err := db.QueryRows[types.Proxy](ctx, "SELECT * FROM proxies", nil)
	if err != nil {
		return nil, err
	}

	return &app{
		entries:    entries,
		kaiaHelper: chainHelper,
		proxies:    proxies,
	}, nil
}

func (a *app) Run(ctx context.Context) {
	go a.cleanupDB(ctx)

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

	log.Debug().Any("entries", a.entries).Msg("launching")

	// start jobs
	wg := sync.WaitGroup{}
	for _, e := range a.entries {
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			a.startJob(ctx, e)
		}(e)
		time.Sleep(100 * time.Millisecond) // prevent sudden requests
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
	v, err := fetcher.FetchSingle(ctx, e.definition, a.buildRequestOpts(e)...)
	if err != nil {
		return err
	}
	log.Debug().Str("entry", e.adapter.Name).Float64("value", v).Msg("fetched")

	defer func() {
		if err := db.QueryWithoutResult(ctx, "INSERT INTO public.por_offchain (name, value) VALUES (@name, @value)", map[string]any{
			"name":  e.adapter.Name,
			"value": v,
		}); err != nil {
			log.Error().Err(err).Msg("failed to insert offchain data")
		}
	}()

	lastInfo, err := a.getLastInfo(ctx, e)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch last info")
	}
	log.Debug().Str("entry", e.adapter.Name).Any("lastInfo", lastInfo).Msg("last info loaded")

	now := time.Now()
	if a.shouldReport(e, lastInfo, v, now) {
		roundId, err := a.getRoundId(ctx, e)
		if err != nil {
			return err
		}
		log.Debug().Str("entry", e.adapter.Name).Uint32("roundId", roundId).Msg("round id loaded")

		err = a.report(ctx, e, v, roundId)
		if err != nil {
			return err
		}
		log.Debug().Str("entry", e.adapter.Name).Msg("reported")
	} else {
		log.Debug().Str("entry", e.adapter.Name).Msg("no need to report")
	}

	return nil
}

func (a *app) report(ctx context.Context, e entry, submissionValue float64, latestRoundId uint32) error {
	tmp := new(big.Float).SetFloat64(submissionValue)
	submissionValueParam := new(big.Int)
	tmp.Int(submissionValueParam)

	latestRoundIdParam := new(big.Int).SetUint64(uint64(latestRoundId))

	err := retrier.Retry(func() error {
		err := a.kaiaHelper.SubmitDirect(
			ctx,
			e.aggregator.Address,
			submitInterface,
			latestRoundIdParam,
			submissionValueParam,
		)
		if err != nil && (chainUtils.IsNonceError(err) || errors.Is(err, context.DeadlineExceeded)) {
			_ = a.kaiaHelper.FlushNoncePool(ctx)
		}
		return err

	}, maxRetry, initialFailureTimeout, maxRetryDelay)

	return err
}

func (a *app) getLastInfo(ctx context.Context, e entry) (lastInfo, error) {
	r := lastInfo{
		UpdatedAt: big.NewInt(0),
		Answer:    big.NewInt(0),
	}

	rawResult, err := a.kaiaHelper.ReadContract(ctx, e.aggregator.Address, latestRoundDataInterface)
	if err != nil {
		return r, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return r, errorSentinel.ErrPorRawResultCastFail
	}

	updatedAt, ok := rawResultSlice[3].(*big.Int)
	if !ok {
		return r, errorSentinel.ErrPorUpdatedAtCastFail
	}

	answer, ok := rawResultSlice[1].(*big.Int)
	if !ok {
		return r, errorSentinel.ErrPorAnswerCastFail
	}

	return lastInfo{
		UpdatedAt: updatedAt,
		Answer:    answer,
	}, nil
}

func (a *app) shouldReport(e entry, lastInfo lastInfo, newVal float64, fetchedTime time.Time) bool {
	if lastInfo.UpdatedAt.Sign() == 0 && lastInfo.Answer.Sign() == 0 {
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

func (a *app) getNextProxy() *string {
	a.Lock()
	defer a.Unlock()

	if len(a.proxies) == 0 {
		return nil
	}

	proxy := a.proxies[a.proxyIdx%len(a.proxies)].GetProxyUrl()
	a.proxyIdx++
	return &proxy
}

func (a *app) buildRequestOpts(e entry) []request.RequestOption {
	opts := []request.RequestOption{}
	if e.useProxy {
		if p := a.getNextProxy(); p != nil && *p != "" {
			opts = append(opts, request.WithProxy(*p))
		}
	}
	return opts
}

func (a *app) cleanupDB(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := db.QueryWithoutResult(ctx, "DELETE FROM public.por_offchain WHERE timestamp < NOW() - INTERVAL '6 hours'", nil); err != nil {
				log.Error().Err(err).Msg("failed to cleanup offchain data")
			}
		}
	}
}
