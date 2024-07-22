package dal

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
	"bisonai.com/orakl/sentinel/pkg/secrets"
	"github.com/rs/zerolog/log"
)

const (
	DefaultDalCheckInterval = 10 * time.Second
	DelayOffset             = 2 * time.Second
)

type OutgoingSubmissionData struct {
	Symbol        string   `json:"symbol"`
	Value         string   `json:"value"`
	AggregateTime string   `json:"aggregateTime"`
	Proof         []byte   `json:"proof"`
	FeedHash      [32]byte `json:"feedHash"`
	Decimals      string   `json:"decimals"`
}

func Start() error {
	interval, err := time.ParseDuration(os.Getenv("DAL_CHECK_INTERVAL"))
	if err != nil {
		interval = DefaultDalCheckInterval
	}

	chain := os.Getenv("CHAIN")
	if chain == "" {
		return errors.New("CHAIN not found")
	}

	key := secrets.GetSecret("DAL_API_KEY")
	if key == "" {
		return errors.New("DAL_API_KEY not found")
	}

	endpoint := fmt.Sprintf("https://dal.%s.orakl.network", chain)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		err := checkDal(endpoint, key)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("error in checkDal")
		}
	}
	return nil
}

func checkDal(endpoint string, key string) error {
	msg := ""

	resp, err := request.Request[[]OutgoingSubmissionData](
		request.WithEndpoint(endpoint+"/latest-data-feeds/all"),
		request.WithHeaders(map[string]string{"X-API-Key": key}),
	)
	if err != nil {
		return err
	}

	for _, data := range resp {
		rawTimestamp, err := strconv.ParseInt(data.AggregateTime, 10, 64)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("failed to convert timestamp string to int64")
			continue
		}

		timestamp := time.Unix(rawTimestamp, 0)
		offset := time.Since(timestamp)
		log.Debug().Str("Player", "DalChecker").Str("symbol", data.Symbol).Time("timestamp", timestamp).Dur("offset", offset).Msg("DAL price check")

		if offset > DelayOffset {
			msg += fmt.Sprintf("(DAL) %s price update delayed by %s\n", data.Symbol, offset)
		}
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}
