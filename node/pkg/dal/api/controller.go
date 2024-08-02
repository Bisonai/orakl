package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func HandleWebsocket(conn *websocket.Conn) {
	h, ok := conn.Locals("hub").(*Hub)
	if !ok {
		log.Error().Msg("hub not found")
		return
	}

	threadSafeClient := NewThreadSafeClient(conn)

	closeHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, text string) error {
		h.unregister <- threadSafeClient
		return closeHandler(code, text)
	})

	ctx, ok := conn.Locals("context").(*context.Context)
	if !ok {
		log.Error().Msg("ctx not found")
		return
	}

	h.register <- threadSafeClient
	apiKey := conn.Headers("X-Api-Key")

	id, err := stats.InsertWebsocketConnection(*ctx, apiKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert websocket connection")
		return
	}
	log.Info().Int32("id", id).Msg("inserted websocket connection")

	defer func() {
		h.unregister <- threadSafeClient
		err = stats.UpdateWebsocketConnection(*ctx, id)
		if err != nil {
			log.Error().Err(err).Msg("failed to update websocket connection")
			return
		}
		log.Info().Int32("id", id).Msg("updated websocket connection")
	}()

	for {
		var msg Subscription
		if err = threadSafeClient.ReadJSON(&msg); err != nil {
			log.Error().Err(err).Msg("failed to read message")
			return
		}

		if msg.Method == "SUBSCRIBE" {
			h.mu.Lock()

			subscriptions, ok := h.clients[threadSafeClient]
			if !ok {
				subscriptions = map[string]bool{}
			}

			valid := []string{}

			for _, param := range msg.Params {
				symbol := strings.TrimPrefix(param, "submission@")
				if _, ok := h.configs[symbol]; !ok {
					continue
				}
				subscriptions[symbol] = true
				valid = append(valid, param)
			}
			h.clients[threadSafeClient] = subscriptions
			h.mu.Unlock()
			err = stats.InsertWebsocketSubscriptions(*ctx, id, valid)
			if err != nil {
				log.Error().Err(err).Msg("failed to insert websocket subscription log")
			}
		}
	}
}

func getSymbols(c *fiber.Ctx) error {
	hub, ok := c.Locals("hub").(*Hub)
	if !ok {
		return errors.New("hub not found")
	}

	result := []string{}
	for key := range hub.configs {
		result = append(result, key)
	}
	return c.JSON(result)
}

func getAllLatestFeeds(c *fiber.Ctx) error {
	collector, ok := c.Locals("collector").(*collector.Collector)
	if !ok {
		return errors.New("collector not found")
	}

	result := collector.GetAllLatestData()
	return c.JSON(result)
}

func getLatestFeeds(c *fiber.Ctx) error {
	collector, ok := c.Locals("collector").(*collector.Collector)
	if !ok {
		return errors.New("collector not found")
	}

	symbolsStr := c.Params("symbols")

	if symbolsStr == "" {
		return errors.New("invalid symbol: empty symbol")
	}

	symbols := strings.Split(symbolsStr, ",")
	results := make([]*dalcommon.OutgoingSubmissionData, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		if !strings.Contains(symbol, "-") {
			return fmt.Errorf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)
		}

		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}

		result, err := collector.GetLatestData(symbol)
		if err != nil {
			return err
		}

		results = append(results, result)
	}

	return c.JSON(results)
}

func getLatestFeedsTransposed(c *fiber.Ctx) error {
	collector, ok := c.Locals("collector").(*collector.Collector)
	if !ok {
		return errors.New("collector not found")
	}

	symbolsStr := c.Params("symbols")

	if symbolsStr == "" {
		return errors.New("invalid symbol: empty symbol")
	}

	symbols := strings.Split(symbolsStr, ",")
	bulk := BulkResponse{}
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		if !strings.Contains(symbol, "-") {
			return fmt.Errorf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)
		}

		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}

		result, err := collector.GetLatestData(symbol)
		if err != nil {
			return err
		}

		bulk.Symbols = append(bulk.Symbols, result.Symbol)
		bulk.Values = append(bulk.Values, result.Value)
		bulk.AggregateTimes = append(bulk.AggregateTimes, result.AggregateTime)
		bulk.Proofs = append(bulk.Proofs, result.Proof)
		bulk.FeedHashes = append(bulk.FeedHashes, result.FeedHash)
		bulk.Decimals = append(bulk.Decimals, result.Decimals)
	}
	return c.JSON(bulk)
}

func getAllLatestFeedsTransposed(c *fiber.Ctx) error {
	collector, ok := c.Locals("collector").(*collector.Collector)
	if !ok {
		return errors.New("collector not found")
	}

	result := collector.GetAllLatestData()
	bulk := BulkResponse{}
	for _, data := range result {
		bulk.Symbols = append(bulk.Symbols, data.Symbol)
		bulk.Values = append(bulk.Values, data.Value)
		bulk.AggregateTimes = append(bulk.AggregateTimes, data.AggregateTime)
		bulk.Proofs = append(bulk.Proofs, data.Proof)
		bulk.FeedHashes = append(bulk.FeedHashes, data.FeedHash)
		bulk.Decimals = append(bulk.Decimals, data.Decimals)
	}
	return c.JSON(bulk)
}
