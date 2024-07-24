package por

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	chainUtils "bisonai.com/orakl/node/pkg/chain/utils"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/fetcher"
	"bisonai.com/orakl/node/pkg/secrets"
	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

func New(ctx context.Context) (*App, error) {
	// TODO: updates for multiple PORs
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

	adapterUrl := "https://config.orakl.network/adapter/" + chain + "/peg-" + chain + ".por.json"
	aggregatorUrl := "https://config.orakl.network/aggregator/" + chain + "/peg.por.json"

	adapter, err := request.Request[AdapterModel](request.WithEndpoint(adapterUrl))
	if err != nil {
		return nil, err
	}

	fetchInterval := 60 * time.Second
	if adapter.Interval != nil {
		fetchInterval = time.Duration(*adapter.Interval) * time.Millisecond
	}

	aggregator, err := request.Request[AggregatorModel](request.WithEndpoint(aggregatorUrl))
	if err != nil {
		return nil, err
	}

	definition := new(fetcher.Definition)
	err = json.Unmarshal(adapter.Feeds[0].Definition, &definition)
	if err != nil {
		return nil, err
	}

	submitInterval := 60 * time.Minute
	if aggregator.Heartbeat != nil {
		submitInterval = time.Duration(*aggregator.Heartbeat) * time.Millisecond
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
		helper.WithoutAdditionalProviderUrls(),
	)
	if err != nil {
		return nil, err
	}

	return &App{
		Name:            adapter.Name,
		Definition:      definition,
		FetchInterval:   fetchInterval,
		SubmitInterval:  submitInterval,
		KaiaHelper:      chainHelper,
		ContractAddress: aggregator.Address,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	go func() {
		port := os.Getenv("POR_PORT")
		if port == "" {
			port = "3000"
		}

		http.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
			// Respond with a simple string
			_, err := w.Write([]byte("Orakl POR"))
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

	ticker := time.NewTicker(a.FetchInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			err := retrier.Retry(
				func() error {
					return a.Execute(ctx)
				},
				MAX_RETRY,
				INITIAL_FAILURE_TIMEOUT,
				MAX_RETRY_DELAY,
			)
			if err != nil {
				log.Error().Err(err).Msg("error in execute")
			}
		}
	}
}

func (a *App) Execute(ctx context.Context) error {
	value, err := a.Fetch(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error in fetch")
	}
	log.Debug().Msg("fetched value")

	lastInfo, err := a.GetLastInfo(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error in read contract")
		return err
	}
	log.Debug().Msg("read last info")

	now := time.Now()
	if a.ShouldReport(&lastInfo, value, now) {
		roundId, err := a.GetRoundID(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to get roundId")
			return err
		}

		err = a.report(ctx, value, roundId)
		if err != nil {
			log.Error().Err(err).Msg("failed to report")
			return err
		}
	} else {
		log.Debug().Msg("no need to report")
	}

	return nil
}

func (a *App) Fetch(ctx context.Context) (float64, error) {
	return fetcher.FetchSingle(ctx, a.Definition)
}

func (a *App) ShouldReport(lastInfo *LastInfo, value float64, fetchedTime time.Time) bool {
	if lastInfo.UpdatedAt.Sign() == 0 && lastInfo.Answer.Sign() == 0 {
		return true
	}

	int64UpdatedAt := lastInfo.UpdatedAt.Int64()
	lastSubmissionTime := time.Unix(int64UpdatedAt, 0)
	log.Debug().Msg("time since last submission: " + fetchedTime.Sub(lastSubmissionTime).String())

	buffer := 5 * time.Second

	if fetchedTime.Sub(lastSubmissionTime) > a.SubmitInterval-buffer {
		return true
	}

	lastSubmittedValue := new(big.Float).SetInt(lastInfo.Answer)
	float64Value, _ := lastSubmittedValue.Float64()

	return a.DeviationCheck(float64Value, value)
}

func (a *App) report(ctx context.Context, submissionValue float64, latestRoundId uint32) error {
	tmp := new(big.Float).SetFloat64(submissionValue)
	submissionValueParam := new(big.Int)
	tmp.Int(submissionValueParam)

	latestRoundIdParam := new(big.Int).SetUint64(uint64(latestRoundId))

	return a.KaiaHelper.SubmitDirect(ctx, a.ContractAddress, SUBMIT_FUNCTION_STRING, latestRoundIdParam, submissionValueParam)
}

func (a *App) DeviationCheck(oldValue float64, newValue float64) bool {
	log.Debug().Float64("oldValue", oldValue).Float64("newValue", newValue).Msg("checking deviation")

	denominator := math.Pow10(DECIMALS)
	old := oldValue / denominator
	new := newValue / denominator

	if old != 0 && new != 0 {
		deviationRange := old * DEVIATION_THRESHOLD
		min := old - deviationRange
		max := old + deviationRange
		return new < min || new > max
	} else if old == 0 && new != 0 {
		return new > ABSOLUTE_THRESHOLD
	} else {
		return false
	}
}

func (a *App) GetRoundID(ctx context.Context) (uint32, error) {
	publicAddress, err := a.KaiaHelper.PublicAddress()
	if err != nil {
		return 0, err
	}

	rawResult, err := a.KaiaHelper.ReadContract(ctx, a.ContractAddress, READ_ROUND_ID, publicAddress, uint32(0))
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

func (a *App) GetLastInfo(ctx context.Context) (LastInfo, error) {
	rawResult, err := a.KaiaHelper.ReadContract(ctx, a.ContractAddress, READ_LATEST_ROUND_DATA)
	if err != nil {
		return LastInfo{}, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return LastInfo{}, errorSentinel.ErrPorRawResultCastFail
	}

	updatedAt, ok := rawResultSlice[3].(*big.Int)
	if !ok {
		return LastInfo{}, errorSentinel.ErrPorUpdatedAtCastFail
	}

	answer, ok := rawResultSlice[1].(*big.Int)
	if !ok {
		return LastInfo{}, errorSentinel.ErrPorAnswerCastFail
	}

	return LastInfo{
		UpdatedAt: updatedAt,
		Answer:    answer,
	}, nil
}
