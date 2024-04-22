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
)

type App struct {
	Name           string
	Definition     *fetcher.Definition
	FetchInterval  time.Duration
	SubmitInterval time.Duration
	KlaytnHelper   *helper.ChainHelper

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
		Name:           adapter.Name,
		Definition:     definition,
		FetchInterval:  fetchInterval,
		SubmitInterval: submitInterval,
		KlaytnHelper:   chainHelper,

		LastSubmissionValue: 0,
		LastSubmissionTime:  time.Time{},
	}, nil
}

/*
POR has different job cycle compared to price feeds
 1. Fetches from the url
 2. Checks if it should be reported
    a. when deviation check returns true
    b. when last submission time is over submission interval
 3. Saves data into its structure and report if required and store in redis for last submitted value & time
*/
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

			if a.ShouldReport(value, time.Now()) {

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
