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
const VRF_EVENT = "vrf_random_words_fulfilled"
const MaxFeedSubmissionCount = 2 // 2 times within the specified feed.ExpectedInterval will be considered oversubmission

var EventCheckInterval time.Duration
var POR_BUFFER = 60 * time.Second
var VRF_ALARM_WINDOW = 180 * time.Second

func setUp(ctx context.Context) (*CheckList, error) {
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
		return nil, err
	}

	pegPorConfig, err := loadPegPorEventInterval()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load peg por event interval")
		return nil, err
	}

	subgraphInfoMap, err := loadSubgraphInfoMap(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load subgraph info")
		return nil, err
	}

	FeedsToCheck := []FeedToCheck{}
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

	pegPorSubgraphInfo, ok := subgraphInfoMap[pegPorConfig.Name]
	if !ok {
		log.Warn().Msg("Peg Por subgraph info not found")
		return nil, fmt.Errorf("por subgraph info not found")
	}

	PegPorToCheck := FeedToCheck{
		SchemaName:       pegPorSubgraphInfo.SchemaName,
		FeedName:         pegPorConfig.Name,
		ExpectedInterval: pegPorConfig.Heartbeat,
		LatencyChecked:   0,
	}

	VRFToCheck := FullfillEventToCheck{
		SchemaName: subgraphInfoMap["VRF"].SchemaName,
		Name:       "VRF Fullfillment",
		EventName:  VRF_EVENT,
	}

	return &CheckList{
		Feeds: FeedsToCheck,
		Por:   PegPorToCheck,
		VRF:   VRFToCheck,
	}, nil

}

func Start(ctx context.Context) error {
	log.Info().Msg("Starting event checker")
	checkList, err := setUp(ctx)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(EventCheckInterval)
	defer ticker.Stop()

	groupedFeeds := groupFeedsByInterval(checkList.Feeds)
	checkGroupedFeeds(ctx, groupedFeeds)

	for range ticker.C {
		checkPorAndVrf(ctx, checkList)
	}
	return nil
}

func groupFeedsByInterval(FeedsToCheck []FeedToCheck) map[int][]FeedToCheck {
	result := make(map[int][]FeedToCheck)
	for i, feed := range FeedsToCheck {
		if result[feed.ExpectedInterval] == nil {
			result[feed.ExpectedInterval] = make([]FeedToCheck, 0)
		}
		result[feed.ExpectedInterval] = append(result[feed.ExpectedInterval], FeedsToCheck[i])
	}
	return result
}

func checkGroupedFeeds(ctx context.Context, feedsByInterval map[int][]FeedToCheck) {
	for interval, feeds := range feedsByInterval {
		ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
		go func(feeds []FeedToCheck) {
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					checkFeeds(ctx, feeds)
				}
			}
		}(feeds)
	}
}

func checkFeeds(ctx context.Context, feedsToCheck []FeedToCheck) {
	msg := ""
	for i := range feedsToCheck {
		msg = checkEachFeed(ctx, &feedsToCheck[i])
	}
	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func checkEachFeed(ctx context.Context, feed *FeedToCheck) string {
	result := ""
	offset, err := timeSinceLastFeedEvent(ctx, *feed)
	if err == nil {
		result += handleFeedSubmissionDelay(offset, feed)
	} else {
		log.Error().Err(err).Str("feed", feed.FeedName).Msg("Failed to check feed")
	}

	count, err := countLastIntervalFeedEvents(ctx, *feed)
	if err == nil {
		result += handleFeedOverSubmission(count, feed)
	} else {
		log.Error().Err(err).Str("feed", feed.FeedName).Msg("Failed to count last minute feed events")
	}
	return result
}

func checkPorAndVrf(ctx context.Context, checkList *CheckList) {
	checkPors(ctx, checkList.Por)
	checkVRF(ctx, checkList.VRF)
}

func checkPors(ctx context.Context, PegPorToCheck FeedToCheck) {
	msg := ""
	porOffset, err := timeSinceLastPorEvent(ctx, PegPorToCheck)
	if err != nil {
		log.Error().Err(err).Str("feed", PegPorToCheck.FeedName).Msg("Failed to check peg por")
	} else {
		log.Debug().Str("POR offset", porOffset.String()).Msg("POR offset")
		if porOffset > time.Duration(PegPorToCheck.ExpectedInterval)*time.Millisecond+POR_BUFFER {
			log.Warn().Str("feed", PegPorToCheck.FeedName).Msg(fmt.Sprintf("%s delayed by %s", PegPorToCheck.FeedName, porOffset-time.Duration(PegPorToCheck.ExpectedInterval)*time.Millisecond))
			msg += fmt.Sprintf("%s delayed by %s\n", PegPorToCheck.FeedName, porOffset-time.Duration(PegPorToCheck.ExpectedInterval)*time.Millisecond)
		}
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func checkVRF(ctx context.Context, vrfToCheck FullfillEventToCheck) {
	msg := ""
	type Fullfillment struct {
		Block int32  `db:"block$"`
		ID    string `db:"id"`
		Time  int64  `db:"time"`
	}
	query := loadUnfullfilledVRFEventQuery(vrfToCheck.SchemaName, vrfToCheck.EventName)
	unfullfilled, err := db.QueryRows[Fullfillment](ctx, query, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query unfullfilled events")
		return
	}
	log.Debug().Msg("Loaded unfullfilled events")

	for _, unfullfilledEvent := range unfullfilled {
		log.Debug().Str("id", unfullfilledEvent.ID).Int32("block", unfullfilledEvent.Block).Str("time", time.Unix(unfullfilledEvent.Time, 0).String()).Msg("Checking unfullfilled event")
		unfullfedTime := time.Unix(unfullfilledEvent.Time, 0)
		offset := time.Since(unfullfedTime)
		// typically takes about 2 seconds to fulfill, skip if less than 3 seconds
		if offset > VRF_ALARM_WINDOW || offset < 3*time.Second {
			continue
		}

		log.Warn().Msg(fmt.Sprintf("%s delayed by %s (id: %s, time: %s)", vrfToCheck.Name, offset, unfullfilledEvent.ID, unfullfedTime.String()))
		msg += fmt.Sprintf("%s delayed by %s (id: %s, time: %s)\n", vrfToCheck.Name, offset, unfullfilledEvent.ID, unfullfedTime.String())
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func timeSinceLastFeedEvent(ctx context.Context, feed FeedToCheck) (time.Duration, error) {
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

func countLastIntervalFeedEvents(ctx context.Context, feed FeedToCheck) (int, error) {
	type Count struct {
		Count int `db:"count"`
	}
	query := feedLastIntervalEventQuery(feed.SchemaName, feed.ExpectedInterval)
	count, err := db.QueryRow[Count](ctx, query, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query last minute event count")
		return 0, err
	}
	return count.Count, nil

}

func timeSinceLastPorEvent(ctx context.Context, feed FeedToCheck) (time.Duration, error) {
	type QueriedTime struct {
		UnixTime int64 `db:"time"`
	}
	query := aggregatorEventQuery(feed.SchemaName)
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
	url := loadOraklConfigUrl(chain)
	return request.Request[[]Config](request.WithEndpoint(url))
}

func loadPegPorEventInterval() (PegPorConfig, error) {
	chain := os.Getenv("CHAIN")
	url := loadPegPorConfigUrl(chain)
	return request.Request[PegPorConfig](request.WithEndpoint(url))
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

		if strings.HasPrefix(subgraphInfo.Name, "Aggregator-") && strings.HasSuffix(subgraphInfo.Name, "-POR") {
			porName := strings.TrimPrefix(subgraphInfo.Name, "Aggregator-")
			subgraphInfoMap[porName] = subgraphInfo
		}

		if subgraphInfo.Name == "VRFCoordinator" {
			subgraphInfoMap["VRF"] = subgraphInfo
		}
	}

	return subgraphInfoMap, nil
}

func handleFeedSubmissionDelay(offset time.Duration, feed *FeedToCheck) string {
	msg := ""
	if offset > time.Duration(feed.ExpectedInterval)*time.Millisecond*2 {
		log.Warn().Str("feed", feed.FeedName).Msg(fmt.Sprintf("%s delayed by %s", feed.FeedName, offset-time.Duration(feed.ExpectedInterval)*time.Millisecond))
		feed.LatencyChecked++
		if feed.LatencyChecked > AlarmOffset {
			msg += fmt.Sprintf("%s delayed by %s\n", feed.FeedName, offset-time.Duration(feed.ExpectedInterval)*time.Millisecond)
			feed.LatencyChecked = 0
		}
	} else {
		feed.LatencyChecked = 0
	}
	return msg
}

func handleFeedOverSubmission(count int, feed *FeedToCheck) string {
	msg := ""
	if count >= MaxFeedSubmissionCount {
		log.Warn().Str("feed", feed.FeedName).Msg(fmt.Sprintf("%s submitted %d times in %d seconds", feed.FeedName, count, feed.ExpectedInterval/1000))
		feed.OversubmissionCount++
		if feed.OversubmissionCount > AlarmOffset {
			msg += fmt.Sprintf("%s made %dx more submissions, %d times consecutively\n", feed.FeedName, MaxFeedSubmissionCount, AlarmOffset+1)
			feed.OversubmissionCount = 0
		}
	} else {
		feed.OversubmissionCount = 0
	}
	return msg
}
