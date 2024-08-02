package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

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
	// wsEndpoint := "ws://localhost:8090/ws"
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

func fetchConfigs() ([]types.Config, error) {

	endpoint := "https://config.orakl.network/baobab_configs.json"
	configs, err := request.Request[[]types.Config](request.WithEndpoint(endpoint))
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

	// log.Info().Any("wsData", wsData).Msg("ws message")

	timestamp, err := strconv.ParseInt(wsData.AggregateTime, 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse timestamp")
		return err
	}

	t := time.Unix(timestamp, 0)

	diff := time.Since(t)
	if diff > time.Second*1 {
		log.Info().Str("Player", "Reporter").Str("Symbol", wsData.Symbol).Str("delay", fmt.Sprintf("%f", diff.Seconds())).Msg("ws message")
	}

	return nil
}
