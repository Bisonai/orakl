package api

import (
	"context"
	"errors"
	"strings"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Setup(ctx context.Context, adminEndpoint string) (*Controller, error) {
	configs, err := request.Request[[]types.Config](request.WithEndpoint(adminEndpoint + "/config"))
	if err != nil {
		log.Error().Err(err).Msg("failed to get configs")
		return nil, err
	}

	configMap := make(map[string]types.Config)
	for _, config := range configs {
		configMap[config.Name] = config
	}
	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("failed to create collector")
		return nil, err
	}

	ApiController := NewController(configMap, collector)
	return ApiController, nil
}

func NewController(configs map[string]types.Config, internalCollector *collector.Collector) *Controller {
	return &Controller{
		Collector: internalCollector,
		configs:   configs,

		clients:    make(map[*websocket.Conn]map[string]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(map[string]chan dalcommon.OutgoingSubmissionData),
	}
}

func (c *Controller) Start(ctx context.Context) {
	go c.Collector.Start(ctx)
	log.Info().Msg("api collector started")
	go func() {
		for {
			select {
			case conn := <-c.register:
				c.clients[conn] = make(map[string]bool)
			case conn := <-c.unregister:
				delete(c.clients, conn)
				conn.Close()
			}
		}
	}()

	for configId, stream := range c.Collector.OutgoingStream {
		symbol := c.configIdToSymbol(configId)
		c.broadcast[symbol] = make(chan dalcommon.OutgoingSubmissionData)
		c.broadcast[symbol] = stream
	}

	for symbol := range c.configs {
		go c.broadcastDataForSymbol(symbol)
	}
}

func (c *Controller) configIdToSymbol(id int32) string {
	for symbol, config := range c.configs {
		if config.ID == id {
			return symbol
		}
	}
	return ""
}

func (c *Controller) broadcastDataForSymbol(symbol string) {
	for data := range c.broadcast[symbol] {
		go c.castSubmissionData(&data, &symbol)
	}
}

// pass by pointer to reduce memory copy time
func (c *Controller) castSubmissionData(data *dalcommon.OutgoingSubmissionData, symbol *string) {
	for conn := range c.clients {
		if _, ok := c.clients[conn][*symbol]; ok {
			if err := conn.WriteJSON(*data); err != nil {
				log.Error().Err(err).Msg("failed to write message")
				delete(c.clients, conn)
				conn.Close()
			}
		}
	}
}

func HandleWebsocket(conn *websocket.Conn) {
	c, ok := conn.Locals("apiController").(*Controller)
	if !ok {
		log.Error().Msg("api controller not found")
		return
	}

	ctx, ok := conn.Locals("context").(*context.Context)
	if !ok {
		log.Error().Msg("ctx not found")
		return
	}

	c.register <- conn
	apiKey := conn.Headers("X-Api-Key")

	id, err := stats.InsertWebsocketConnection(*ctx, apiKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert websocket connection")
		return
	}
	log.Info().Int32("id", id).Msg("inserted websocket connection")

	defer func() {
		c.unregister <- conn
		conn.Close()
		err := stats.UpdateWebsocketConnection(*ctx, id)
		if err != nil {
			log.Error().Err(err).Msg("failed to update websocket connection")
			return
		}
		log.Info().Int32("id", id).Msg("updated websocket connection")
	}()

	for {
		var msg Subscription
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error().Err(err).Msg("failed to read message")
			return
		}

		if msg.Method == "SUBSCRIBE" {

			if c.clients[conn] == nil {
				c.clients[conn] = make(map[string]bool)
			}
			for _, param := range msg.Params {
				symbol := strings.TrimPrefix(param, "submission@")
				if _, ok := c.configs[symbol]; !ok {
					continue
				}
				c.clients[conn][symbol] = true
				err = stats.InsertWebsocketSubscription(*ctx, id, param)
				if err != nil {
					log.Error().Err(err).Msg("failed to insert websocket subscription")
				}
			}
		}
	}
}

func getSymbols(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
	}

	result := []string{}
	for key := range controller.configs {
		result = append(result, key)
	}
	return c.JSON(result)
}

func getLatestFeeds(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
	}

	result := controller.Collector.GetAllLatestData()
	return c.JSON(result)
}

func getLatestFeed(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
	}

	symbol := c.Params("symbol")

	if symbol == "" {
		return errors.New("invalid symbol: empty symbol")
	}
	if !strings.Contains(symbol, "-") {
		return errors.New("symbol should be in {BASE}-{QUOTE} format")
	}

	if !strings.Contains(symbol, "test") {
		symbol = strings.ToUpper(symbol)
	}

	result, err := controller.Collector.GetLatestData(symbol)
	if err != nil {
		return err
	}

	return c.JSON(*result)
}
