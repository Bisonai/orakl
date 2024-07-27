package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Setup(ctx context.Context, adminEndpoint string) (*Controller, error) {
	configs, err := request.Request[[]types.Config](request.WithEndpoint(adminEndpoint + "/config"))
	if err != nil {
		log.Error().Str("Player", "controller").Err(err).Msg("failed to get configs")
		return nil, err
	}

	configMap := make(map[string]types.Config)
	for _, config := range configs {
		configMap[config.Name] = config
	}
	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Str("Player", "controller").Err(err).Msg("failed to create collector")
		return nil, err
	}

	ApiController := NewController(configMap, collector)
	return ApiController, nil
}

func NewController(configs map[string]types.Config, internalCollector *collector.Collector) *Controller {
	maxConns, err := strconv.Atoi(os.Getenv("MAX_CONN"))
	if err != nil {
		maxConns = DEFAULT_MAX_CONNS
	}

	return &Controller{
		Collector: internalCollector,
		configs:   configs,

		clients:    make(map[*websocket.Conn]map[string]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(map[string]chan dalcommon.OutgoingSubmissionData),
		connQueue:  make(map[string][]*websocket.Conn),
		maxConns:   maxConns,
		mu:         sync.RWMutex{},
	}
}

func (c *Controller) Start(ctx context.Context) {
	go c.Collector.Start(ctx)
	log.Info().Str("Player", "controller").Msg("api collector started")
	go c.handleConnection(ctx)
	log.Info().Str("Player", "controller").Msg("connection handler started")
	go c.startBroadCast()
}

func (c *Controller) handleConnection(ctx context.Context) {
	for {
		select {
		case conn := <-c.register:
			c.mu.Lock()
			if _, ok := c.clients[conn]; !ok {
				c.clients[conn] = make(map[string]bool)
				c.addConQueue(conn.IP(), conn)
			}
			c.mu.Unlock()
		case conn := <-c.unregister:
			c.mu.Lock()
			if _, ok := c.clients[conn]; ok {
				c.removeConQueue(conn.IP(), conn)
				delete(c.clients, conn)
			}
			c.mu.Unlock()
		case <-ctx.Done():
			c.mu.Lock()
			for conn := range c.clients {
				delete(c.clients, conn)
			}
			c.mu.Unlock()
			return
		}
	}
}

func (c *Controller) addConQueue(ip string, conn *websocket.Conn) {
	if conns, ok := c.connQueue[ip]; ok {
		if len(conns) >= c.maxConns {
			oldest := conns[0]
			go func(oldest *websocket.Conn) {
				log.Info().Str("Player", "controller").Str("IP", ip).Msg("too many connections, closing oldest")
				c.unregister <- oldest
				_ = oldest.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"), time.Now().Add(time.Second))
				oldest.Close()
			}(oldest)
			c.connQueue[ip] = conns[1:] // Remove oldest from the queue
		}
		c.connQueue[ip] = append(conns, conn)
	} else {
		c.connQueue[ip] = []*websocket.Conn{conn}
	}
}

func (c *Controller) removeConQueue(ip string, conn *websocket.Conn) {
	if conns, ok := c.connQueue[ip]; ok {
		for i, entry := range conns {
			if entry == conn {
				c.connQueue[ip] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(c.connQueue[ip]) == 0 {
			delete(c.connQueue, ip)
		}
	}
}

func (c *Controller) startBroadCast() {
	for configId, stream := range c.Collector.OutgoingStream {
		symbol := c.configIdToSymbol(configId)
		if symbol == "" {
			continue
		}
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
		go c.castSubmissionData(&data, symbol)
	}
}

func (c *Controller) castSubmissionData(data *dalcommon.OutgoingSubmissionData, symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conn := range c.clients {
		if _, ok := c.clients[conn][symbol]; ok {
			if err := conn.WriteJSON(*data); err != nil {
				log.Error().Str("Player", "controller").Err(err).Msg("failed to write message")
				c.unregister <- conn
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "failed to write message"), time.Now().Add(time.Second))
				conn.Close()
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

func getAllLatestFeeds(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
	}

	result := controller.Collector.GetAllLatestData()
	return c.JSON(result)
}

func getLatestFeeds(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
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

		result, err := controller.Collector.GetLatestData(symbol)
		if err != nil {
			return err
		}

		results = append(results, result)
	}

	return c.JSON(results)
}

func getLatestFeedsTransposed(c *fiber.Ctx) error {
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
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

		result, err := controller.Collector.GetLatestData(symbol)
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
	controller, ok := c.Locals("apiController").(*Controller)
	if !ok {
		return errors.New("api controller not found")
	}

	result := controller.Collector.GetAllLatestData()
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
