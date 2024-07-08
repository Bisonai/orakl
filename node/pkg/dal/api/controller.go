package api

import (
	"context"
	"errors"
	"strings"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

var ApiController Controller

func Setup(ctx context.Context) error {
	configs, err := db.QueryRows[types.Config](ctx, "SELECT * FROM configs", nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to get configs")
		return err
	}
	configMap := make(map[string]types.Config)
	for _, config := range configs {
		configMap[config.Name] = config
	}
	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("failed to create collector")
		return err
	}

	ApiController = *NewController(configMap, collector)
	return nil
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

func (c *Controller) handleWebsocket(conn *websocket.Conn) {
	c.register <- conn
	defer func() {
		c.unregister <- conn
		conn.Close()
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
			}
		}
	}
}

func (c *Controller) getLatestSubmissionData(ctx context.Context) ([]aggregator.SubmissionData, error) {
	globalAggregateKeyList := make([]string, 0, len(c.configs))
	for _, config := range c.configs {
		globalAggregateKeyList = append(globalAggregateKeyList, keys.GlobalAggregateKey(config.ID))
	}

	globalAggregates, err := db.MGetObject[aggregator.GlobalAggregate](ctx, globalAggregateKeyList)
	if err != nil {
		return nil, err
	}

	proofKeyList := make([]string, 0, len(globalAggregates))
	for _, globalAggregate := range globalAggregates {
		proofKeyList = append(proofKeyList, keys.ProofKey(globalAggregate.ConfigID, globalAggregate.Round))
	}

	proofs, err := db.MGetObject[aggregator.Proof](ctx, proofKeyList)
	if err != nil {
		return nil, err
	}

	proofMap := make(map[int32]aggregator.Proof)
	for _, proof := range proofs {
		proofMap[proof.ConfigID] = proof
	}

	result := make([]aggregator.SubmissionData, 0, len(globalAggregates))
	for _, globalAggregate := range globalAggregates {
		proof, ok := proofMap[globalAggregate.ConfigID]
		if !ok {
			continue
		}

		result = append(result, aggregator.SubmissionData{
			GlobalAggregate: globalAggregate,
			Proof:           proof,
		})
	}

	return result, nil
}

func (c *Controller) getLatestSubmissionDataSingle(ctx context.Context, symbol string) (*aggregator.SubmissionData, error) {
	config, ok := c.configs[symbol]
	if !ok {
		return nil, errors.New("invalid symbol")
	}

	globalAggregate, err := db.GetObject[aggregator.GlobalAggregate](ctx, keys.GlobalAggregateKey(config.ID))
	if err != nil {
		return nil, err
	}

	proof, err := db.GetObject[aggregator.Proof](ctx, keys.ProofKey(config.ID, globalAggregate.Round))
	if err != nil {
		return nil, err
	}

	return &aggregator.SubmissionData{
		GlobalAggregate: globalAggregate,
		Proof:           proof,
	}, nil
}

func getLatestFeeds(c *fiber.Ctx) error {
	submissionData, err := ApiController.getLatestSubmissionData(c.Context())
	if err != nil {
		return err
	}

	result := make([]dalcommon.OutgoingSubmissionData, 0, len(submissionData))

	for _, data := range submissionData {
		outgoingData, err := ApiController.Collector.IncomingDataToOutgoingData(c.Context(), data)
		if err != nil {
			return err
		}
		result = append(result, *outgoingData)
	}
	return c.JSON(result)
}

func getLatestFeed(c *fiber.Ctx) error {
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

	submissionData, err := ApiController.getLatestSubmissionDataSingle(c.Context(), symbol)
	if err != nil {
		return err
	}

	outgoingData, err := ApiController.Collector.IncomingDataToOutgoingData(c.Context(), *submissionData)
	if err != nil {
		return err
	}
	return c.JSON(outgoingData)
}
