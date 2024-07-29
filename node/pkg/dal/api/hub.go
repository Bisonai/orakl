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
		configs:    configs,
		clients:    make(map[*ThreadSafeClient]map[string]bool),
		register:   make(chan *ThreadSafeClient),
		unregister: make(chan *ThreadSafeClient),
		broadcast:  make(map[string]chan dalcommon.OutgoingSubmissionData),
		connPerIP:  make(map[string][]*websocket.Conn),
	}
}

func (h *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go h.handleClientRegistration()

	h.initializeBroadcastChannels(collector)

	for symbol := range h.configs {
		go h.broadcastDataForSymbol(symbol)
	}
}

func (h *Hub) handleClientRegistration() {
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
	c.mu.Lock() // Use write lock for both checking and insertion
	defer c.mu.Unlock()
	if _, ok := c.clients[client]; ok {
		return
	}
	c.clients[client] = make(map[string]bool)
	
	if _, ok := h.connPerIP[conn.IP()]; !ok {
		h.connPerIP[conn.IP()] = make([]*websocket.Conn, 0)
	}

	h.connPerIP[conn.IP()] = append(h.connPerIP[conn.IP()], conn)
	
	if len(h.connPerIP) > MAX_CONNECTIONS {
		oldConn := h.connPerIP[conn.IP()][0]
		if subs, ok := h.clients[oldConn]; ok {
			for k := range subs {
				delete(h.clients[oldConn], k)
			}
		}
		delete(h.clients, oldConn)
		oldConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"),
			time.Now().Add(time.Second),
		)
		oldConn.Close()
	}
}

func (c *Hub) removeClient(client *ThreadSafeClient) {
	c.mu.Lock() // Use write lock for both checking and removal
	defer c.mu.Unlock()
	subscriptions, ok := c.clients[client]
	if !ok {
		return
	}
	delete(c.clients, client)
	for symbol := range subscriptions {
		delete(subscriptions, symbol)
	}

	for i, c := range h.connPerIP[conn.IP()] {
		if c == conn {
			h.connPerIP[conn.IP()] = append(h.connPerIP[conn.IP()][:i], h.connPerIP[conn.IP()][i+1:]...)
			if len(h.connPerIP) == 0 {
				delete(h.connPerIP, conn.IP())
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

func (h *Hub) initializeBroadcastChannels(collector *collector.Collector) {
	for configId, stream := range collector.OutgoingStream {
		symbol := c.configIdToSymbol(configId)
		if symbol == "" {
			continue
		}

		c.broadcast[symbol] = stream
	}
}

func (h *Hub) configIdToSymbol(id int32) string {
	for symbol, config := range h.configs {
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

	c.mu.Lock()
	defer c.mu.Unlock()
	for client, subscriptions := range c.clients {
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
	}
	wg.Wait()
}
