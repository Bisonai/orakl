package api

import (
	"context"
	"strings"

	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

func HandleWebsocket(conn *websocket.Conn) {
	c, ok := conn.Locals("apiController").(*Controller)
	if !ok {
		log.Error().Str("Player", "controller").Msg("api controller not found")
		return
	}

	closeHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, text string) error {
		log.Info().Str("Player", "controller").Int("code", code).Str("text", text).Msg("close handler")
		c.unregister <- conn
		return closeHandler(code, text)
	})

	ctxPointer, ok := conn.Locals("context").(*context.Context)
	if !ok {
		log.Error().Str("Player", "controller").Msg("ctx not found")
		return
	}

	ctx := *ctxPointer
	apiKey := conn.Headers("X-Api-Key")

	c.register <- conn

	id, err := stats.InsertWebsocketConnection(ctx, apiKey)
	if err != nil {
		log.Error().Str("Player", "controller").Err(err).Msg("failed to insert websocket connection")
		return
	}
	log.Info().Str("Player", "controller").Int32("id", id).Msg("inserted websocket connection")

	defer func() {
		conn.CloseHandler()(websocket.CloseNormalClosure, "normal closure")
		err = stats.UpdateWebsocketConnection(ctx, id)
		if err != nil {
			log.Error().Str("Player", "controller").Err(err).Msg("failed to update websocket connection")
			return
		}
		log.Info().Str("Player", "controller").Int32("id", id).Msg("updated websocket connection")
	}()

	for {
		if err := handleMessage(ctx, conn, c, id); err != nil {
			log.Error().Str("Player", "controller").Err(err).Msg("failed to handle message")
			return
		}
	}
}

func handleMessage(ctx context.Context, conn *websocket.Conn, c *Controller, id int32) error {
	var msg Subscription
	if err := conn.ReadJSON(&msg); err != nil {
		return err
	}

	if msg.Method == "SUBSCRIBE" {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.clients[conn] == nil {
			c.clients[conn] = make(map[string]bool)
		}
		addedSymbols := make([]string, 0, len(msg.Params))
		for _, param := range msg.Params {
			symbol := strings.TrimPrefix(param, "submission@")
			if _, ok := c.configs[symbol]; !ok {
				continue
			}

			if _, ok := c.clients[conn][symbol]; ok {
				continue
			}

			c.clients[conn][symbol] = true
			addedSymbols = append(addedSymbols, symbol)
		}
		defer func() {
			err := stats.InsertWebsocketSubscriptions(ctx, id, addedSymbols)
			if err != nil {
				log.Error().Str("Player", "controller").Err(err).Msg("failed to insert websocket subscriptions")
			}
		}()
	}

	return nil
}
