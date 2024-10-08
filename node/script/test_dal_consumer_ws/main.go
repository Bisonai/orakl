package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

type Config = types.Config

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func main() {
	ctx := context.Background()
	chain := "baobab"
	key := ""
	configs, err := fetchConfigs()
	if err != nil {
		panic(err)
	}
	subscription := Subscription{
		Method: "SUBSCRIBE",
		Params: []string{},
	}

	for _, configs := range configs {
		subscription.Params = append(subscription.Params, "submission@"+configs.Name)
	}

	wsEndpoint := fmt.Sprintf("ws://dal.%s.orakl.network/ws", chain)

	wsHelper, err := wss.NewWebsocketHelper(
		ctx,
		wss.WithEndpoint(wsEndpoint),
		wss.WithSubscriptions([]any{subscription}),
		wss.WithRequestHeaders(map[string]string{"X-API-Key": key}),
	)
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go wsHelper.Run(ctx, handleWsMessage)
	wg.Wait()
}

func fetchConfigs() ([]Config, error) {

	endpoint := "https://config.orakl.network/baobab_configs.json"
	configs, err := request.Request[[]Config](request.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return configs, nil
}

type WsResponse struct {
	Symbol        string    `json:"symbol"`
	AggregateTime string    `json:"aggregateTime"`
	PublishTime   time.Time `json:"publishTime"`
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

	timestamp, err := strconv.ParseInt(wsData.AggregateTime, 10, 64)
	if err != nil {
		log.Error().Err(err).Str("data", string(jsonMarshalData)).Msg("failed to parse timestamp")
		return err
	}

	t := time.UnixMilli(timestamp)

	diff := time.Since(t)
	if diff > time.Second {
		log.Info().Str("Player", "Reporter").Str("Symbol", wsData.Symbol).Str("delay", fmt.Sprintf("%f", diff.Seconds())).Msg("ws message")
	}

	return nil
}
