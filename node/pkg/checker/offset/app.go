package offset

// track offset of latest local aggregate & global aggregate
// if offset exceeds threshold, will send an alarm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"slices"

	"bisonai.com/miko/node/pkg/alert"
	"bisonai.com/miko/node/pkg/checker"
	"bisonai.com/miko/node/pkg/db"
	nodemetrics "bisonai.com/miko/node/pkg/metrics"
	"bisonai.com/miko/node/pkg/secrets"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/rs/zerolog/log"
)

const (
	AlarmOffsetPerPair = 10
	AlarmOffsetInTotal = 3

	PriceDifferenceThreshold = 0.1
	DelayThreshold           = 5 * time.Second
	DefaultCheckInterval     = 5 * time.Minute

	nodeMetricsEndpoint = "/metrics"
	nodeMetricsTimeout  = 10 * time.Second

	PorAlarmThreshold     = 3
	PorAlarmOffset        = 300 * time.Second
	GetPorOffchainOffsets = `SELECT
	name,
	EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - MAX(timestamp))) AS delay
FROM
	por_offchain
GROUP BY
	name`
)

// aggregateMetrics holds the latest aggregates of one config read from node
// metrics; nil means the series is absent.
type aggregateMetrics struct {
	LocalValue      *float64
	LocalTimestamp  *float64
	GlobalValue     *float64
	GlobalTimestamp *float64
}

type PorOffsetResult struct {
	Name  string
	Delay float64
}

var localAggregateAlarmCount map[string]int
var globalAggregateAlarmCount map[string]int
var porOffchainAlarmCount map[string]int

func Start(ctx context.Context) error {
	localAggregateAlarmCount = make(map[string]int)
	globalAggregateAlarmCount = make(map[string]int)
	porOffchainAlarmCount = make(map[string]int)

	serviceDBUrl := secrets.GetSecret("SERVICE_DB_URL")
	if serviceDBUrl == "" {
		log.Error().Msg("Missing SERVICE_DB_URL")
		return errors.New("missing SERVICE_DB_URL")
	}

	serviceDB, err := db.GetTransientPool(ctx, serviceDBUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error getting service DB connection")
		return err
	}
	defer serviceDB.Close()

	ticker := time.NewTicker(DefaultCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context cancelled, shutting down")
			return ctx.Err()
		case <-ticker.C:
			go checkOffsets(ctx)
			go checkPorOffsets(ctx, serviceDB)
		}
	}
}

func checkPorOffsets(ctx context.Context, serviceDB *pgxpool.Pool) {
	msg := ""

	res, err := db.QueryRowsTransient[PorOffsetResult](ctx, serviceDB, GetPorOffchainOffsets, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting por offchain offsets")
		return
	}

	for _, r := range res {
		log.Debug().Str("name", r.Name).Float64("delay", r.Delay).Msg("checking por offset")
		if r.Delay > PorAlarmOffset.Seconds() {
			porOffchainAlarmCount[r.Name]++
			if porOffchainAlarmCount[r.Name] > PorAlarmThreshold {
				msg += fmt.Sprintf("(por offchain offset) %s: %v seconds\n", r.Name, r.Delay)
				porOffchainAlarmCount[r.Name] = 0
			}
		} else {
			porOffchainAlarmCount[r.Name] = 0
		}
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func checkOffsets(ctx context.Context) {
	nodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if nodeAdminUrl == "" {
		log.Error().Msg("ORAKL_NODE_ADMIN_URL not found")
		return
	}

	aggregates, err := fetchAggregateMetrics(ctx, nodeAdminUrl+nodeMetricsEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Error getting node aggregate metrics")
		return
	}

	msg := offsetAlarmMessage(aggregates, time.Now())
	if msg != "" {
		alert.SlackAlert(msg)
	}
}

func offsetAlarmMessage(aggregates map[string]*aggregateMetrics, now time.Time) string {
	msg := ""

	names := make([]string, 0, len(aggregates))
	for name := range aggregates {
		names = append(names, name)
	}
	slices.Sort(names)

	nowSeconds := float64(now.UnixNano()) / 1e9
	localAggregateDelayedOffsetCount := 0
	globalAggregateDelayedOffsetCount := 0
	for _, name := range names {
		if slices.Contains(checker.SymbolsToBeDelisted, name) {
			continue
		}
		aggregate := aggregates[name]

		log.Debug().Str("name", name).Msg("checking config offset")
		if aggregate.LocalTimestamp != nil {
			delay := nowSeconds - *aggregate.LocalTimestamp
			if delay > DelayThreshold.Seconds() {
				localAggregateAlarmCount[name]++
				if localAggregateAlarmCount[name] > AlarmOffsetPerPair {
					msg += fmt.Sprintf("(local aggregate offset delayed) %s: %v seconds\n", name, delay)
					localAggregateAlarmCount[name] = 0
				} else if localAggregateAlarmCount[name] > AlarmOffsetInTotal {
					localAggregateDelayedOffsetCount++
				}
			} else {
				localAggregateAlarmCount[name] = 0
			}
		}

		if aggregate.GlobalTimestamp != nil {
			delay := nowSeconds - *aggregate.GlobalTimestamp
			if delay > DelayThreshold.Seconds() {
				globalAggregateAlarmCount[name]++
				if globalAggregateAlarmCount[name] > AlarmOffsetPerPair {
					msg += fmt.Sprintf("(global aggregate offset delayed) %s: %v seconds\n", name, delay)
					globalAggregateAlarmCount[name] = 0
				} else if globalAggregateAlarmCount[name] > AlarmOffsetInTotal {
					globalAggregateDelayedOffsetCount++
				}
			} else {
				globalAggregateAlarmCount[name] = 0
			}
		}

		if aggregate.LocalValue == nil || aggregate.GlobalValue == nil {
			continue
		}
		localAggregate := *aggregate.LocalValue
		globalAggregate := *aggregate.GlobalValue
		if math.Abs(localAggregate-globalAggregate)/localAggregate > PriceDifferenceThreshold {
			msg += fmt.Sprintf("(latest aggregate price difference) %s: %v (globalAggregate: %v, localAggregate: %v)\n", name, localAggregate-globalAggregate, globalAggregate, localAggregate)
		}
	}

	if localAggregateDelayedOffsetCount > 0 {
		msg += fmt.Sprintf("%d local aggregate offsets delayed\n", localAggregateDelayedOffsetCount)
	}

	if globalAggregateDelayedOffsetCount > 0 {
		msg += fmt.Sprintf("%d global aggregate offsets delayed\n", globalAggregateDelayedOffsetCount)
	}

	return msg
}

func fetchAggregateMetrics(ctx context.Context, url string) (map[string]*aggregateMetrics, error) {
	ctx, cancel := context.WithTimeout(ctx, nodeMetricsTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	return parseAggregateMetrics(resp.Body)
}

func parseAggregateMetrics(r io.Reader) (map[string]*aggregateMetrics, error) {
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(r)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[string]*aggregateMetrics)
	collect := func(metricName string, field func(*aggregateMetrics) **float64) {
		family, ok := families[metricName]
		if !ok {
			return
		}
		for _, metric := range family.GetMetric() {
			name := ""
			for _, label := range metric.GetLabel() {
				if label.GetName() == nodemetrics.NameLabel {
					name = label.GetValue()
				}
			}
			if name == "" || metric.GetGauge() == nil {
				continue
			}
			if aggregates[name] == nil {
				aggregates[name] = &aggregateMetrics{}
			}
			value := metric.GetGauge().GetValue()
			*field(aggregates[name]) = &value
		}
	}

	collect(nodemetrics.LocalAggregateValueName, func(a *aggregateMetrics) **float64 { return &a.LocalValue })
	collect(nodemetrics.LocalAggregateTimestampName, func(a *aggregateMetrics) **float64 { return &a.LocalTimestamp })
	collect(nodemetrics.GlobalAggregateValueName, func(a *aggregateMetrics) **float64 { return &a.GlobalValue })
	collect(nodemetrics.GlobalAggregateTimestampName, func(a *aggregateMetrics) **float64 { return &a.GlobalTimestamp })

	return aggregates, nil
}
