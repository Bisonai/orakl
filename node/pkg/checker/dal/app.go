package dal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"bisonai.com/miko/node/pkg/checker"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/configfetch"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

func (u *UpdateTimes) Store(symbol string, time time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastUpdates[symbol] = time
}

// exportLastUpdates publishes the last WebSocket push time per symbol.
func (u *UpdateTimes) exportLastUpdates() {
	u.mu.RLock()
	defer u.mu.RUnlock()

	for symbol, updateTime := range u.lastUpdates {
		if slices.Contains(checker.SymbolsToBeDelisted, symbol) {
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
		err := checkDal(endpoint, key, restSymbols)
		if err != nil {
			restErrorsCounter.Inc()
			log.Error().Str("Player", "DalChecker").Err(err).Msg("error in checkDal")
		}
		updateTimes.exportLastUpdates()
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

// checkDal exports the aggregate time of every DAL REST entry. restSymbols
// holds the symbols exported last round; symbols no longer served are dropped
// so they don't look stale forever.
func checkDal(endpoint string, key string, restSymbols map[string]struct{}) error {
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

	for symbol := range restSymbols {
		if _, ok := seen[symbol]; !ok {
			restAggregateTimestampGauge.DeleteLabelValues(symbol)
			delete(restSymbols, symbol)
		}
	}
	for symbol := range seen {
		restSymbols[symbol] = struct{}{}
	}

	restEmptyFeedsGauge.Set(float64(empty))
	restLastSuccessGauge.Set(float64(time.Now().Unix()))
	return nil
}

func isDataEmpty(data *OutgoingSubmissionData) bool {
	return data.Symbol == "" || data.Value == "" || data.AggregateTime == "" || data.Proof == "" || data.FeedHash == "" || data.Decimals == ""
}

func fetchConfigs() ([]Config, error) {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Info().Str("Player", "Reporter").Msg("CHAIN env not set, defaulting to kairos")
		chain = "kairos"
	}
	return configfetch.Get[[]Config](context.Background(), configfetch.FeedsPath(chain))
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
