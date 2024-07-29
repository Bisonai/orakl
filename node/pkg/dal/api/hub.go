package api

import (
	"context"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

func HubSetup(ctx context.Context, configs []types.Config) *Hub {
	configMap := make(map[string]types.Config)
	for _, config := range configs {
		configMap[config.Name] = config
	}

	hub := NewHub(configMap)
	return hub
}

func NewHub(configs map[string]types.Config) *Hub {
	return &Hub{
		configs: configs,

		clients:    make(map[*websocket.Conn]map[string]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(map[string]chan dalcommon.OutgoingSubmissionData),
	}
}

func (c *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go func() {
		for {
			select {
			case conn := <-c.register:
				c.mu.Lock()
				c.clients[conn] = make(map[string]bool)
				c.mu.Unlock()
			case conn := <-c.unregister:
				c.mu.Lock()
				delete(c.clients, conn)
				conn.Close()
				c.mu.Unlock()
			}
		}
	}()

	for configId, stream := range collector.OutgoingStream {
		symbol := c.configIdToSymbol(configId)
		if symbol == "" {
			continue
		}
		c.broadcast[symbol] = make(chan dalcommon.OutgoingSubmissionData)
		c.broadcast[symbol] = stream
	}

	for symbol := range c.configs {
		go c.broadcastDataForSymbol(symbol)
	}
}

func (c *Hub) configIdToSymbol(id int32) string {
	for symbol, config := range c.configs {
		if config.ID == id {
			return symbol
		}
	}
	return ""
}

func (c *Hub) broadcastDataForSymbol(symbol string) {
	for data := range c.broadcast[symbol] {

		go c.castSubmissionData(&data, &symbol)
	}
}

// pass by pointer to reduce memory copy time
func (c *Hub) castSubmissionData(data *dalcommon.OutgoingSubmissionData, symbol *string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for conn := range c.clients {
		if _, ok := c.clients[conn][*symbol]; ok {
			if err := conn.WriteJSON(*data); err != nil {
				log.Error().Err(err).Msg("failed to write message")
				c.unregister <- conn
			}
		}
	}
}
