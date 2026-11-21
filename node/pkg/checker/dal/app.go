package dal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/checker"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/configfetch"
	"bisonai.com/miko/node/pkg/utils/configreload"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

func (u *UpdateTimes) Store(symbol string, time time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastUpdates[symbol] = time
}

// exportLastUpdates publishes the last WebSocket push time per expected
// symbol. A symbol that is no longer expected (removed from config on reload,
// or delisted) is dropped together with its series, so its frozen push time
// does not keep the WS staleness alert firing.
func (u *UpdateTimes) exportLastUpdates(expected map[string]struct{}) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for symbol, updateTime := range u.lastUpdates {
		if _, ok := expected[symbol]; !ok {
			wsLastPushGauge.DeleteLabelValues(symbol)
			delete(u.lastUpdates, symbol)
			continue
		}
		wsLastPushGauge.WithLabelValues(symbol).Set(float64(updateTime.UnixMilli()) / 1e3)
	}
}

func Start(ctx context.Context) error {
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

	var endpoint, wsEndpoint string
	// In-cluster gate: testnet uses the in-cluster DAL. Gate on the normalized
	// value so it keeps selecting in-cluster after a CHAIN flip baobab->kairos.
	if chain == "kairos" {
		endpoint = "http://orakl-dal.orakl.svc.cluster.local"
		wsEndpoint = "ws://orakl-dal.orakl.svc.cluster.local/ws"
	} else {
		// External host is REAL external DNS (not a served config name), so the
		// raw chain is used here unchanged: CHAIN=mainnet yields
		// dal.mainnet.orakl.network (live; the mainnet DAL runs in KakaoCloud).
		// The DAL_CHECK_* overrides below remain a seam for infra to set a
		// different host.
		endpoint = fmt.Sprintf("https://dal.%s.orakl.network", chain)
		wsEndpoint = fmt.Sprintf("ws://dal.%s.orakl.network/ws", chain)
	}
	// Phase-4 infra seam: if set, these overrides win verbatim over the
	// chain-based construction above (and over the in-cluster default).
	if v := os.Getenv("DAL_CHECK_REST_URL"); v != "" {
		endpoint = v
	}
	if v := os.Getenv("DAL_CHECK_WS_URL"); v != "" {
		wsEndpoint = v
	}

	configs, err := fetchConfigs()
	if err != nil {
		return err
	}
	expected := &expectedSymbols{}
	expected.set(configs)
	// Follow config reloads (the DAL reloads the same bundle) so a feed
	// removed from config is not reported missing until a restart.
	reloader := configreload.New(configfetch.FeedsPath(configChain()), len(configs), func(c Config) string { return c.Name },
		func(_ context.Context, c []Config) error {
			expected.set(c)
			return nil
		})
	go reloader.Run(ctx, configreload.Interval(), nil)

	subscription := Subscription{
		Method: "SUBSCRIBE",
		Params: buildSubscriptionParams(configs),
	}

	wsHelper, err := wss.NewWebsocketHelper(
		ctx,
		wss.WithEndpoint(wsEndpoint),
		wss.WithSubscriptions([]interface{}{subscription}),
		wss.WithRequestHeaders(map[string]string{"X-API-Key": key}),
	)
	if err != nil {
		return err
	}

	go wsHelper.Run(ctx, handleWsMessage)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Alerts: Prometheus rules on the exported orakl_dal_check_* metrics.
	restSymbols := map[string]struct{}{}
	for range ticker.C {
		current := expected.get()
		err := checkDal(endpoint, key, current, restSymbols)
		if err != nil {
			restErrorsCounter.Inc()
			log.Error().Str("Player", "DalChecker").Err(err).Msg("error in checkDal")
		}
		updateTimes.exportLastUpdates(current)
	}
	return nil
}

func buildSubscriptionParams(configs []Config) []string {
	params := make([]string, len(configs))
	for i, config := range configs {
		params[i] = "submission@" + config.Name
	}
	return params
}

// expectedSymbols is the configured (non-delisted) symbol set, replaced on
// config reload.
type expectedSymbols struct {
	mu      sync.RWMutex
	symbols map[string]struct{}
}

func (e *expectedSymbols) set(configs []Config) {
	symbols := make(map[string]struct{}, len(configs))
	for _, c := range configs {
		if slices.Contains(checker.SymbolsToBeDelisted, c.Name) {
			continue
		}
		symbols[c.Name] = struct{}{}
	}
	e.mu.Lock()
	e.symbols = symbols
	e.mu.Unlock()
}

// get returns the current set; callers must not modify it.
func (e *expectedSymbols) get() map[string]struct{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.symbols
}

// checkDal exports the aggregate time of every DAL REST entry and the number
// of expected (configured) symbols missing from the response. restSymbols
// holds the symbols exported so far. A configured symbol that is not served
// keeps its last series so staleness alerts fire; only symbols that are
// neither served nor configured are dropped. A response without any valid
// feed is a failed check and leaves the series untouched.
func checkDal(endpoint string, key string, expected map[string]struct{}, restSymbols map[string]struct{}) error {
	now := time.Now()
	resp, err := request.Request[[]OutgoingSubmissionData](
		request.WithEndpoint(endpoint+"/latest-data-feeds/all"),
		request.WithHeaders(map[string]string{"X-API-Key": key}),
		request.WithTimeout(RestTimeout),
	)
	restDurationGauge.Set(time.Since(now).Seconds())
	if err != nil {
		return err
	}

	seen := map[string]struct{}{}
	empty := 0
	for _, data := range resp {
		if slices.Contains(checker.SymbolsToBeDelisted, data.Symbol) {
			continue
		}
		if isDataEmpty(&data) {
			log.Debug().Str("Player", "DalChecker").Any("data", data).Msg("data is empty")
			empty++
		}
		if data.Symbol == "" {
			continue
		}

		rawTimestamp, err := strconv.ParseInt(data.AggregateTime, 10, 64)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("failed to convert timestamp string to int64")
			continue
		}
		log.Debug().Str("Player", "DalChecker").Str("symbol", data.Symbol).Dur("offset", time.Since(time.UnixMilli(rawTimestamp))).Msg("DAL price check")

		restAggregateTimestampGauge.WithLabelValues(data.Symbol).Set(float64(rawTimestamp) / 1e3)
		seen[data.Symbol] = struct{}{}
	}

	var missing []string
	for symbol := range expected {
		if _, ok := seen[symbol]; !ok {
			missing = append(missing, symbol)
		}
	}
	restMissingFeedsGauge.Set(float64(len(missing)))
	if len(missing) > 0 {
		slices.Sort(missing)
		log.Warn().Str("Player", "DalChecker").Int("count", len(missing)).Str("symbols", strings.Join(missing, ",")).Msg("configured feeds missing from DAL REST response or with invalid aggregateTime")
	}

	restEmptyFeedsGauge.Set(float64(empty))
	if len(seen) == 0 {
		// Failed check: keep all series as they are.
		return fmt.Errorf("DAL REST response has no valid feeds (%d entries)", len(resp))
	}

	for symbol := range restSymbols {
		if _, ok := seen[symbol]; ok {
			continue
		}
		if _, ok := expected[symbol]; ok {
			continue
		}
		restAggregateTimestampGauge.DeleteLabelValues(symbol)
		delete(restSymbols, symbol)
	}
	for symbol := range seen {
		restSymbols[symbol] = struct{}{}
	}

	restLastSuccessGauge.Set(float64(time.Now().Unix()))
	return nil
}

func isDataEmpty(data *OutgoingSubmissionData) bool {
	return data.Symbol == "" || data.Value == "" || data.AggregateTime == "" || data.Proof == "" || data.FeedHash == "" || data.Decimals == ""
}

func fetchConfigs() ([]Config, error) {
	return configfetch.Get[[]Config](context.Background(), configfetch.FeedsPath(configChain()))
}

func configChain() string {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Info().Str("Player", "Reporter").Msg("CHAIN env not set, defaulting to kairos")
		chain = "kairos"
	}
	return chain
}

func handleWsMessage(ctx context.Context, data map[string]interface{}) error {
	wsData := WsResponse{}
	jsonMarshalData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonMarshalData, &wsData)
	if err != nil {
		return err
	}
	updateTimes.Store(wsData.Symbol, time.Now())
	return nil
}
