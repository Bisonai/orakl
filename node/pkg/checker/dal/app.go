package dal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/secrets"
	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

const (
	DefaultDalCheckInterval = 10 * time.Second
	DelayOffset             = 5 * time.Second
	AlarmOffset             = 3
	WsDelayThreshold        = 9 * time.Second
	WsPushThreshold         = 5 * time.Second
)

var (
	wsChan      = make(chan WsResponse, 30000)
	wsMsgChan   = make(chan string, 10000)
	updateTimes = &UpdateTimes{
		lastUpdates: make(map[string]time.Time),
	}
	re = regexp.MustCompile(`\(([^)]+)\)`)
)

type WsResponse struct {
	Symbol        string `json:"symbol"`
	AggregateTime string `json:"aggregateTime"`
}

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Config struct {
	Name           string `json:"name"`
	SubmitInterval *int   `json:"submitInterval"`
}

type OutgoingSubmissionData struct {
	Symbol        string `json:"symbol"`
	Value         string `json:"value"`
	AggregateTime string `json:"aggregateTime"`
	Proof         string `json:"proof"`
	FeedHash      string `json:"feedHash"`
	Decimals      string `json:"decimals"`
}

type UpdateTimes struct {
	lastUpdates map[string]time.Time
	mu          sync.RWMutex
}

func (u *UpdateTimes) Store(symbol string, time time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastUpdates[symbol] = time
}

func (u *UpdateTimes) CheckLastUpdateOffsets(alarmCount map[string]int) []string {
	u.mu.RLock()
	defer u.mu.RUnlock()

	var messages []string
	for symbol, updateTime := range u.lastUpdates {
		elapsedTime := time.Since(updateTime)
		if elapsedTime > WsPushThreshold {
			alarmCount[symbol]++
			if alarmCount[symbol] > AlarmOffset {
				message := fmt.Sprintf("(%s) WebSocket not pushed for %v seconds", symbol, elapsedTime.Seconds())
				messages = append(messages, message)
				alarmCount[symbol] = 0
			}
		} else {
			alarmCount[symbol] = 0
		}
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
	go filterDelayedWsResponse()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	alarmCount := map[string]int{}
	wsPushAlarmCount := map[string]int{}
	wsDelayAlarmCount := map[string]int{}

	for range ticker.C {
		err := checkDal(endpoint, key, alarmCount)
		if err != nil {
			log.Error().Str("Player", "DalChecker").Err(err).Msg("error in checkDal")
		}
		log.Debug().Msg("checking DAL WebSocket")
		checkDalWs(ctx, wsPushAlarmCount, wsDelayAlarmCount)
		log.Debug().Msg("checked DAL WebSocket")
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

func checkDal(endpoint string, key string, alarmCount map[string]int) error {
	msg := ""

	now := time.Now()
	resp, err := request.Request[[]OutgoingSubmissionData](
		request.WithEndpoint(endpoint+"/latest-data-feeds/all"),
		request.WithHeaders(map[string]string{"X-API-Key": key}),
	)
	networkDelay := time.Since(now)

	if err != nil {
		return err
	}

	for _, data := range resp {
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
			if alarmCount[data.Symbol] > AlarmOffset {
				msg += fmt.Sprintf("(DAL) %s price update delayed by %s\n", data.Symbol, offset)
				alarmCount[data.Symbol] = 0
			}
		} else {
			alarmCount[data.Symbol] = 0
		}

	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}

func checkDalWs(ctx context.Context, wsPushAlarmCount, wsDelayAlarmCount map[string]int) {
	log.Debug().Msg("checking WebSocket message delays")

	if msgs := extractWsDelayAlarms(ctx, wsDelayAlarmCount); len(msgs) > 0 {
		alert.SlackAlert(strings.Join(msgs, "\n"))
	}

	log.Debug().Msg("checking WebSocket message push")
	if msgs := updateTimes.CheckLastUpdateOffsets(wsPushAlarmCount); len(msgs) > 0 {
		alert.SlackAlert(strings.Join(msgs, "\n"))
	}
}

func extractWsDelayAlarms(ctx context.Context, alarmCount map[string]int) []string {
	log.Debug().Msg("extracting WebSocket alarms")

	var rawMsgs = []string{}

	select {
	case <-ctx.Done():
		return nil
	case entry := <-wsMsgChan:
		rawMsgs = append(rawMsgs, entry)
	loop:
		for {
			select {
			case entry := <-wsMsgChan:
				rawMsgs = append(rawMsgs, entry)
			default:
				break loop
			}
		}
	default:
		return nil
	}

	delayedSymbols := map[string]any{}
	resultMsgs := []string{}
	for _, entry := range rawMsgs {
		match := re.FindStringSubmatch(entry)
		symbol := match[1]
		delayedSymbols[symbol] = struct{}{}
		alarmCount[symbol]++
		if alarmCount[symbol] > AlarmOffset {
			resultMsgs = append(resultMsgs, entry)
			alarmCount[symbol] = 0
		}
	}

	for symbol := range alarmCount {
		if _, exists := delayedSymbols[symbol]; !exists {
			alarmCount[symbol] = 0
		}
	}

	return resultMsgs
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
	defer updateTimes.Store(wsData.Symbol, time.Now())
	wsChan <- wsData
	return nil
}

func filterDelayedWsResponse() {
	log.Debug().Msg("filtering WebSocket responses")
	for entry := range wsChan {
		timestamp, err := strconv.ParseInt(entry.AggregateTime, 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse timestamp for WebSocket response")
			continue
		}

		if diff := time.Since(time.UnixMilli(timestamp)); diff > WsDelayThreshold {
			wsMsgChan <- fmt.Sprintf("(%s) ws delayed by %v sec", entry.Symbol, diff.Seconds())
		}
	}
}
