package api

import (
	"context"
	"sync"
	"time"

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

		register:   make(chan *ThreadSafeClient),
		unregister: make(chan *ThreadSafeClient),
		broadcast:  make(map[string]chan dalcommon.OutgoingSubmissionData),
	}
}

func (c *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go c.handleClientRegistration()

	c.initializeBroadcastChannels(collector)

	for symbol := range c.configs {
		go c.broadcastDataForSymbol(symbol)
	}
}

func (c *Hub) handleClientRegistration() {
	for {
		select {
		case client := <-c.register:
			c.addClient(client)
		case client := <-c.unregister:
			c.removeClient(client)
		}
	}
}

func (c *Hub) addClient(client *ThreadSafeClient) {
	c.clients.LoadOrStore(client, make(map[string]bool))
}

func (c *Hub) removeClient(client *ThreadSafeClient) {
	raw, ok := c.clients.LoadAndDelete(client)
	if ok {
		subscriptions, typeOk := raw.(map[string]bool)
		if typeOk {
			for symbol := range subscriptions {
				delete(subscriptions, symbol)
			}
		}
	}

	err := client.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed to write close message")
	}
	err = client.Close()
	if err != nil {
		log.Warn().Err(err).Msg("failed to close connection")
	}
}

func (c *Hub) initializeBroadcastChannels(collector *collector.Collector) {
	for configId, stream := range collector.OutgoingStream {
		symbol := c.configIdToSymbol(configId)
		c.broadcast[symbol] = stream
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
	var wg sync.WaitGroup
	c.clients.Range(func(threadSafeClient, symbols any) bool {
		client, ok := threadSafeClient.(*ThreadSafeClient)
		if !ok {
			return true
		}

		subscriptions := symbols.(map[string]bool)
		if subscriptions[*symbol] {
			wg.Add(1)
			go func(entry *ThreadSafeClient) {
				defer wg.Done()
				if err := entry.WriteJSON(*data); err != nil {
					log.Error().Err(err).Msg("failed to write message")
					c.unregister <- entry
				}
			}(client)
		}
		return true
	})
	wg.Wait()
}
