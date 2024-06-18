package event

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/db"
	"bisonai.com/orakl/sentinel/pkg/request"
	"github.com/rs/zerolog/log"
)

const AlarmOffset = 3

var FeedsToCheck = []FeedToCheck{}
var EventCheckInterval time.Duration
var BUFFER = 1 * time.Second

func setUp(ctx context.Context) error {
	log.Debug().Msg("Setting up event checker")
	EventCheckInterval = 60 * time.Second

	interval := os.Getenv("EVENT_CHECK_INTERVAL")
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse EVENT_CHECK_INTERVAL, using default 60s")
	} else {
		EventCheckInterval = parsedInterval
	}

	configs, err := loadExpectedEventIntervals()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load expected event intervals")
		return err
	}
	subgraphInfoMap, err := loadSubgraphInfoMap(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load subgraph info")
		return err
	}

	for _, config := range configs {
		subgraphInfo, ok := subgraphInfoMap[config.Name]
		if !ok {
			continue
		}

		if subgraphInfo.Status != "current" {
			continue
		}

		FeedsToCheck = append(FeedsToCheck, FeedToCheck{
			SchemaName:       subgraphInfo.SchemaName,
			FeedName:         config.Name,
			ExpectedInterval: config.SubmitInterval,
			LatencyChecked:   0,
		})
	}

	return nil
}

func Start(ctx context.Context) error {
	log.Info().Msg("Starting event checker")
	err := setUp(ctx)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(EventCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		check(ctx)
	}
	return nil
}

func check(ctx context.Context) {
	msg := ""
	for i, feed := range FeedsToCheck {
		offset, err := timeSinceLastEvent(ctx, feed)
		if err != nil {
			log.Error().Err(err).Str("feed", feed.FeedName).Msg("Failed to check feed")
			continue
		}

		if offset > time.Duration(feed.ExpectedInterval)*time.Millisecond+BUFFER {
			log.Warn().Str("feed", feed.FeedName).Msg(fmt.Sprintf("%s delayed by %s\n", feed.FeedName, offset-time.Duration(feed.ExpectedInterval)*time.Millisecond))
			feed.LatencyChecked++
			if feed.LatencyChecked > AlarmOffset {
				msg += fmt.Sprintf("%s delayed by %s\n", feed.FeedName, offset-time.Duration(feed.ExpectedInterval)*time.Millisecond)
				feed.LatencyChecked = 0
			}
		} else {
			feed.LatencyChecked = 0
		}
		FeedsToCheck[i] = feed
	}
	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func timeSinceLastEvent(ctx context.Context, feed FeedToCheck) (time.Duration, error) {
	type QueriedTime struct {
		UnixTime int64 `db:"time"`
	}
	query := feedEventQuery(feed.SchemaName)
	queriedTime, err := db.QueryRow[QueriedTime](ctx, query, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query last event time")
		return 0, err
	}
	lastEventTime := time.Unix(queriedTime.UnixTime, 0)
	return time.Since(lastEventTime), nil
}

func loadExpectedEventIntervals() ([]Config, error) {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}
	url := loadOraklConfigUrl(chain)
	return request.GetRequest[[]Config](url, nil, nil)
}

func loadSubgraphInfoMap(ctx context.Context) (map[string]SubgraphInfo, error) {
	subgraphInfos, err := db.QueryRows[SubgraphInfo](ctx, SubgraphInfoQuery, nil)
	if err != nil {
		return nil, err
	}

	subgraphInfoMap := make(map[string]SubgraphInfo)
	for _, subgraphInfo := range subgraphInfos {
		if strings.HasPrefix(subgraphInfo.Name, "Feed-") {
			pricePairName := strings.TrimPrefix(subgraphInfo.Name, "Feed-")
			subgraphInfoMap[pricePairName] = subgraphInfo
		}
	}

	return subgraphInfoMap, nil
}
