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
		connPerIP:  make(map[string][]*ThreadSafeClient),
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
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *ThreadSafeClient) {
	h.mu.Lock() // Use write lock for both checking and insertion
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		return
	}
	h.clients[client] = make(map[string]bool)

	ip := client.Conn.IP()
	if _, ok := h.connPerIP[ip]; !ok {
		h.connPerIP[ip] = make([]*ThreadSafeClient, 0)
	}

	h.connPerIP[ip] = append(h.connPerIP[ip], client)
	if len(h.connPerIP[ip]) > MAX_CONNECTIONS {
		oldConn := h.connPerIP[ip][0]

		subscriptions, ok := h.clients[oldConn]
		if !ok {
			return
		}
		delete(h.clients, oldConn)
		for symbol := range subscriptions {
			delete(subscriptions, symbol)
		}
		h.connPerIP[ip] = h.connPerIP[ip][1:]
		oldConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"),
			time.Now().Add(time.Second),
		)
		oldConn.Close()
	}
}

func (h *Hub) removeClient(client *ThreadSafeClient) {
	h.mu.Lock() // Use write lock for both checking and removal
	defer h.mu.Unlock()
	subscriptions, ok := h.clients[client]
	if !ok {
		return
	}
	delete(h.clients, client)
	for symbol := range subscriptions {
		delete(subscriptions, symbol)
	}

	ip := client.Conn.IP()

	for i, entry := range h.connPerIP[ip] {
		if entry == client {
			h.connPerIP[ip] = append(h.connPerIP[ip][:i], h.connPerIP[ip][i+1:]...)
			if len(h.connPerIP) == 0 {
				delete(h.connPerIP, ip)
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
		symbol := h.configIdToSymbol(configId)
		if symbol == "" {
			continue
		}

		h.broadcast[symbol] = stream
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

func (h *Hub) broadcastDataForSymbol(symbol string) {
	for data := range h.broadcast[symbol] {
		go h.castSubmissionData(&data, &symbol)
	}
}

// pass by pointer to reduce memory copy time
func (c *Hub) castSubmissionData(data *dalcommon.OutgoingSubmissionData, symbol *string) {
	var wg sync.WaitGroup
	clientsToNotify := make([]*ThreadSafeClient, 0)

	c.mu.RLock()
	for client, subscriptions := range c.clients {
		if subscriptions[*symbol] {
			clientsToNotify = append(clientsToNotify, client)
		}
	}
	c.mu.RUnlock()

	for _, client := range clientsToNotify {
		wg.Add(1)
		go func(entry *ThreadSafeClient) {
			defer wg.Done()
			if err := entry.WriteJSON(*data); err != nil {
				log.Error().Err(err).Msg("failed to write message")
				c.unregister <- entry
			}
		}(client)
	}
	wg.Wait()
}
