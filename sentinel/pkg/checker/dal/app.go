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

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
	"bisonai.com/orakl/sentinel/pkg/secrets"
	wss "bisonai.com/orakl/sentinel/pkg/ws"
	"github.com/rs/zerolog/log"
)

const (
	DefaultDalCheckInterval = 10 * time.Second
	DelayOffset             = 5 * time.Second
	AlarmOffset             = 3

	WsDelayThreshold = 9 * time.Second
	WsPushThreshold  = 5 * time.Second
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

var wsChan = make(chan WsResponse, 30000)
var wsMsgChan = make(chan string, 10000)
var updateTimes = &UpdateTimes{
	lastUpdates: make(map[string]time.Time),
}
var re = regexp.MustCompile(`\(([^)]+)\)`)

func (u *UpdateTimes) Store(symbol string, time time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastUpdates[symbol] = time
}

func (u *UpdateTimes) CheckLastUpdateOffsets(pushAlarmCount map[string]int) []string {
	u.mu.RLock()
	defer u.mu.RUnlock()

	msgsNotRecieved := []string{}
	for symbol, updateTime := range u.lastUpdates {
		diff := time.Since(updateTime)
		if diff > WsPushThreshold {
			pushAlarmCount[symbol]++
			if pushAlarmCount[symbol] > AlarmOffset {
				msg := fmt.Sprintf("(%s) ws not pushed for %v(sec)", symbol, diff.Seconds())
				msgsNotRecieved = append(msgsNotRecieved, msg)
				pushAlarmCount[symbol] = 0
			}
		}
	}
	return msgsNotRecieved
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
		Params: []string{},
	}

	for _, configs := range configs {
		subscription.Params = append(subscription.Params, "submission@"+configs.Name)
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
	go filterWsReponses()

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

		timestamp := time.Unix(rawTimestamp, 0)
		offset := time.Since(timestamp)
		log.Debug().Str("Player", "DalChecker").Dur("network delay", networkDelay).Str("symbol", data.Symbol).Time("timestamp", timestamp).Dur("offset", offset).Msg("DAL price check")

		if checkValueEmptiness(&data) {
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

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	msgs := extractWsAlarms(ctxWithTimeout, wsDelayAlarmCount)
	if len(msgs) > 0 {
		alert.SlackAlert(strings.Join(msgs, "\n"))
	}

	log.Debug().Msg("checking WebSocket message push")
	msgsNotRecieved := updateTimes.CheckLastUpdateOffsets(wsPushAlarmCount)
	if len(msgsNotRecieved) > 0 {
		alert.SlackAlert(strings.Join(msgsNotRecieved, "\n"))
	}
}

func extractWsAlarms(ctx context.Context, alarmCount map[string]int) []string {
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
	}

	resultMsgs := []string{}
	for _, entry := range rawMsgs {
		match := re.FindStringSubmatch(entry)
		symbol := match[1]

		alarmCount[symbol]++
		if alarmCount[symbol] > AlarmOffset {
			resultMsgs = append(resultMsgs, entry)
			alarmCount[symbol] = 0
		}
	}

	return resultMsgs
}

func checkValueEmptiness(data *OutgoingSubmissionData) bool {
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

func filterWsReponses() {
	log.Debug().Msg("filtering WebSocket responses")
	for entry := range wsChan {
		strTimestamp := entry.AggregateTime

		unixTimestamp, err := strconv.ParseInt(strTimestamp, 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse timestamp for WebSocket response")
			continue
		}

		timestamp := time.Unix(unixTimestamp, 0)
		diff := time.Since(timestamp)
		if diff > WsDelayThreshold {
			wsMsgChan <- fmt.Sprintf("(%s) ws delayed by %v(sec)", entry.Symbol, diff.Seconds())
		}
	}
}
