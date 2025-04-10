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
	"time"

	"bisonai.com/miko/node/pkg/alert"
	"bisonai.com/miko/node/pkg/checker"
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func (u *UpdateTimes) Store(symbol string, time time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastUpdates[symbol] = time
}

func (u *UpdateTimes) CheckLastUpdateOffsets(alarmCount map[string]int) []string {
	u.mu.RLock()
	defer u.mu.RUnlock()

	websocketNotPushedCount := 0
	var messages []string
	for symbol, updateTime := range u.lastUpdates {
		if slices.Contains(checker.SymbolsToBeDelisted, symbol) {
			continue
		}

		elapsedTime := time.Since(updateTime)
		if elapsedTime > WsPushThreshold {
			alarmCount[symbol]++
			if alarmCount[symbol] > AlarmOffsetPerPair {
				message := fmt.Sprintf("(%s) Websocket not pushed for %v seconds", symbol, elapsedTime.Seconds())
				messages = append(messages, message)
				alarmCount[symbol] = 0
			} else if alarmCount[symbol] > AlarmOffsetInTotal {
				websocketNotPushedCount++
			}
		} else {
			alarmCount[symbol] = 0
		}
	}

	if websocketNotPushedCount > 0 {
		messages = append(messages, fmt.Sprintf("Websocket not being pushed for %d symbols", websocketNotPushedCount))
	}

	return messages
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

	endpoint := fmt.Sprintf("https://dal.%s.orakl.network", chain)
	wsEndpoint := fmt.Sprintf("ws://dal.%s.orakl.network/ws", chain)

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

	dalDBConnectionUrl := secrets.GetSecret("DAL_DB_CONNECTION_URL")
	if dalDBConnectionUrl == "" {
		return errors.New("DAL_DB_CONNECTION_URL not found")
	}

	pool, err := db.GetTransientPool(ctx, dalDBConnectionUrl)
	if err != nil {
		return err
	}
	defer pool.Close()

	networkDelayAlarmCount := 0
	alarmCount := map[string]int{}
	wsDelayAlarmCount := map[string]int{}

	for range ticker.C {
		err := checkDal(endpoint, key, alarmCount, &networkDelayAlarmCount)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("error in checkDal")
		}
		log.Debug().Msg("checking DAL WebSocket")
		checkDalWs(wsDelayAlarmCount)
		log.Debug().Msg("checked DAL WebSocket")

		if err := checkDalTraffic(ctx, pool); err != nil {
			log.Error().Err(err).Msg("error in checkDalTraffic")
		}
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

func checkDal(endpoint string, key string, alarmCount map[string]int, networkDelayAlarmCount *int) error {
	msg := ""

	now := time.Now()
	resp, err := request.Request[[]OutgoingSubmissionData](
		request.WithEndpoint(endpoint+"/latest-data-feeds/all"),
		request.WithHeaders(map[string]string{"X-API-Key": key}),
		request.WithTimeout(RestTimeout),
	)
	networkDelay := time.Since(now)

	if networkDelay > NetworkDelayThreshold {
		*networkDelayAlarmCount++
		if *networkDelayAlarmCount > AlarmOffsetInTotal {
			msg += fmt.Sprintf("(DAL) network delay: %s\n", networkDelay)
			*networkDelayAlarmCount = 0
		}
	} else {
		*networkDelayAlarmCount = 0
	}

	if err != nil {
		return err
	}

	totalDelayed := 0
	for _, data := range resp {
		if slices.Contains(checker.SymbolsToBeDelisted, data.Symbol) {
			continue
		}

		rawTimestamp, err := strconv.ParseInt(data.AggregateTime, 10, 64)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("failed to convert timestamp string to int64")
			continue
		}

		offset := time.Since(time.UnixMilli(rawTimestamp))
		log.Debug().Str("Player", "DalChecker").Dur("network delay", networkDelay).Str("symbol", data.Symbol).Dur("offset", offset).Msg("DAL price check")

		if isDataEmpty(&data) {
			log.Debug().Str("Player", "DalChecker").Msg("data is empty")
			msg += fmt.Sprintf("(DAL) empty data exists among data\n %v\n", data)
		}

		if offset > DelayOffset+networkDelay {
			alarmCount[data.Symbol]++
			if alarmCount[data.Symbol] > AlarmOffsetPerPair {
				msg += fmt.Sprintf("(DAL) %s price update delayed by %s\n", data.Symbol, offset)
				alarmCount[data.Symbol] = 0
			} else if alarmCount[data.Symbol] > AlarmOffsetInTotal {
				totalDelayed++
			}
		} else {
			alarmCount[data.Symbol] = 0
		}

	}

	if totalDelayed > 0 {
		msg += fmt.Sprintf("DAL price update delayed by %d symbols\n", totalDelayed)
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}

func checkDalWs(wsPushAlarmCount map[string]int) {
	log.Debug().Msg("checking WebSocket message push")
	if msgs := updateTimes.CheckLastUpdateOffsets(wsPushAlarmCount); len(msgs) > 0 {
		alert.SlackAlert(strings.Join(msgs, "\n"))
	}
}

func isDataEmpty(data *OutgoingSubmissionData) bool {
	return data.Symbol == "" || data.Value == "" || data.AggregateTime == "" || data.Proof == "" || data.FeedHash == "" || data.Decimals == ""
}

func fetchConfigs() ([]Config, error) {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Info().Str("Player", "Reporter").Msg("CHAIN env not set, defaulting to baobab")
		chain = "baobab"
	}
	endpoint := fmt.Sprintf("https://config.orakl.network/%s_configs.json", chain)
	configs, err := request.Request[[]Config](request.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return configs, nil
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

func checkDalTraffic(ctx context.Context, pool *pgxpool.Pool) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		prev, err := db.QueryRowTransient[Count](ctx, pool, getTrafficCheckQuery(TrafficOldOffset), map[string]any{})
		if err != nil {
			log.Error().Err(err).Msg("failed to check DAL traffic")
			return err
		}
		recent, err := db.QueryRowTransient[Count](ctx, pool, getTrafficCheckQuery(TrafficRecentOffset), map[string]any{})
		if err != nil {
			log.Error().Err(err).Msg("failed to check DAL traffic")
			return err
		}
		if recent.Count > prev.Count {
			alert.SlackAlert(fmt.Sprintf("DAL traffic alert: last 10seconds call count %d exceeded last 10 minutes call count %d", recent.Count, prev.Count))
		}
		return nil
	}
}

func getTrafficCheckQuery(offset int) string {
	keysToIgnore := strings.Split(IgnoreKeys, ",")
	modified := []string{}
	for _, desc := range keysToIgnore {
		modified = append(modified, fmt.Sprintf("'%s'", desc))
	}

	return fmt.Sprintf(TrafficCheckQuery, offset, strings.Join(modified, ", "))
}
