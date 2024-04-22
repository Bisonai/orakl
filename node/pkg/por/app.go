package por

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/fetcher"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

const (
	DECIMALS            = 4
	DEVIATION_THRESHOLD = 0.0001
	ABSOLUTE_THRESHOLD  = 0.1

	INITIAL_FAILURE_TIMEOUT = 50 * time.Millisecond
	MAX_RETRY               = 3
	MAX_RETRY_DELAY         = 500 * time.Millisecond

	SUBMIT_FUNCTION_STRING = "submit(uint256 _roundId, int256 _submission)"
	READ_ROUND_ID          = `function oracleRoundState(address _oracle, uint32 _queriedRoundId) external view returns (
            bool _eligibleToSubmit,
            uint32 _roundId,
            int256 _latestSubmission,
            uint64 _startedAt,
            uint64 _timeout,
            uint8 _oracleCount
    )`
)

type App struct {
	Name            string
	Definition      *fetcher.Definition
	FetchInterval   time.Duration
	SubmitInterval  time.Duration
	KlaytnHelper    *helper.ChainHelper
	ContractAddress string

	LastSubmissionValue float64
	LastSubmissionTime  time.Time
}

type FeedModel struct {
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
	AdapterId  *int64          `json:"adapterId"`
}

type AdapterModel struct {
	Name     string      `json:"name"`
	Feeds    []FeedModel `json:"feeds"`
	Interval *int        `json:"interval"`
}

type AggregatorModel struct {
	Name      string `json:"name"`
	Heartbeat *int   `json:"heartbeat"`
	Address   string `json:"address"`
}

type SubmissionModel struct {
	Name  string    `json:"name"`
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

func New(ctx context.Context) (*App, error) {
	// TODO: updates for multiple PORs
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}

	adapterUrl := "https://raw.githubusercontent.com/Bisonai/orakl-config/master/adapter/" + chain + "/peg-" + chain + ".por.json"
	aggregatorUrl := "https://raw.githubusercontent.com/Bisonai/orakl-config/master/aggregator/" + chain + "/peg.por.json"

	adapter, err := request.GetRequest[AdapterModel](adapterUrl, nil, nil)
	if err != nil {
		return nil, err
	}

	fetchInterval := 60 * time.Second
	if adapter.Interval != nil {
		fetchInterval = time.Duration(*adapter.Interval) * time.Millisecond
	}

	aggregator, err := request.GetRequest[AggregatorModel](aggregatorUrl, nil, nil)
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

	chainHelper, err := helper.NewKlayHelper(ctx, "")
	if err != nil {
		return nil, err
	}

	return &App{
		Name:            adapter.Name,
		Definition:      definition,
		FetchInterval:   fetchInterval,
		SubmitInterval:  submitInterval,
		KlaytnHelper:    chainHelper,
		ContractAddress: aggregator.Address,

		LastSubmissionValue: 0,
		LastSubmissionTime:  time.Time{},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.FetchInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			value, err := fetcher.FetchSingle(ctx, a.Definition)
			if err != nil {
				log.Error().Err(err).Msg("error in fetch")
			}

			if a.LastSubmissionTime.IsZero() && a.LastSubmissionValue == 0 {
				loaded, err := LoadSubmission(ctx, a.Name)
				if err == nil {
					a.LastSubmissionTime = loaded.Time
					a.LastSubmissionValue = loaded.Value
				}
			}

			now := time.Now()
			if a.ShouldReport(value, now) {
				err := a.report(ctx, now, value)
				if err != nil {
					log.Error().Err(err).Msg("failed to report")
					continue
				}
			}
		}
	}
}

func (a *App) ShouldReport(value float64, fetchedTime time.Time) bool {
	if a.LastSubmissionTime.IsZero() && a.LastSubmissionValue == 0 {
		return true
	}

	if fetchedTime.Sub(a.LastSubmissionTime) > a.SubmitInterval {
		return true
	}

	if a.DeviationCheck(a.LastSubmissionValue, value) {
		return true
	}
	return false
}

func (a *App) report(ctx context.Context, submissionTime time.Time, submissionValue float64) error {
	reportJob := func() error {
		latestRoundId, err := a.ReadLatestRoundId(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to read latest round id")
			return err
		}

		tx, err := a.KlaytnHelper.MakeDirectTx(ctx, a.ContractAddress, SUBMIT_FUNCTION_STRING, latestRoundId, submissionValue)
		if err != nil {
			log.Error().Err(err).Msg("failed to make direct tx")
			return err
		}

		err = a.KlaytnHelper.SubmitRawTx(ctx, tx)
		if err != nil {
			log.Error().Err(err).Msg("failed to submit raw tx")
			return err
		}
		return nil
	}

	err := retry(reportJob)
	if err != nil {
		log.Error().Err(err).Msg("failed to report")
		return err
	}
	a.LastSubmissionTime = submissionTime
	a.LastSubmissionValue = submissionValue
	return StoreSubmission(ctx, a.Name, submissionValue, submissionTime)
}

func (a *App) DeviationCheck(oldValue float64, newValue float64) bool {
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

func (a *App) ReadLatestRoundId(ctx context.Context) (uint32, error) {
	publicAddress, err := a.KlaytnHelper.PublicAddress()
	if err != nil {
		return 0, err
	}

	rawResult, err := a.KlaytnHelper.ReadContract(ctx, a.ContractAddress, READ_ROUND_ID, publicAddress, uint32(0))
	if err != nil {
		return 0, err
	}

	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return 0, errors.New("failed to cast raw result to slice")
	}

	result, ok := rawResultSlice[1].(uint32)
	if !ok {
		return 0, errors.New("failed to cast roundId to uint32")
	}

	return result, nil
}

func StoreSubmission(ctx context.Context, name string, value float64, time time.Time) error {
	key := "POR:" + name
	data, err := json.Marshal(SubmissionModel{Name: name, Value: value, Time: time})
	if err != nil {
		return err
	}
	return db.Set(ctx, key, string(data), 0)
}

// loaded only when not found within app structure
func LoadSubmission(ctx context.Context, name string) (SubmissionModel, error) {
	key := "POR:" + name
	var result SubmissionModel
	data, err := db.Get(ctx, key)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func retry(job func() error) error {
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

func calculateJitter(baseTimeout time.Duration) time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to generate jitter for retry timeout")
		return baseTimeout
	}
	jitter := time.Duration(n.Int64()) * time.Millisecond
	return baseTimeout + jitter
}
