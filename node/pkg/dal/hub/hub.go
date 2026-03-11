package hub

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/collector"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/dal/utils/stats"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Hub struct {
	Symbols    map[string]struct{}
	Clients    map[*websocket.Conn]map[string]struct{}
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	broadcast  map[string]chan *dalcommon.OutgoingSubmissionData
	mu         sync.RWMutex
}

const (
	MAX_CONNECTIONS = 10
	CleanupInterval = time.Hour
	WriteTimeout    = 10 * time.Second
)

func HubSetup(ctx context.Context, configs []types.Config) *Hub {
	symbolsMap := make(map[string]struct{})
	for _, config := range configs {
		symbolsMap[config.Name] = struct{}{}
	}

	hub := NewHub(symbolsMap)
	return hub
}

func NewHub(symbols map[string]struct{}) *Hub {
	return &Hub{
		Symbols:    symbols,
		Clients:    make(map[*websocket.Conn]map[string]struct{}),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
		broadcast:  make(map[string]chan *dalcommon.OutgoingSubmissionData),
	}
}

func (h *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go h.handleClientRegistration(ctx)

	h.initializeBroadcastChannels(collector)

	for symbol := range h.Symbols {
		sym := symbol // Capture loop variable to avoid potential race condition
		go h.broadcastDataForSymbol(ctx, sym)
	}

	go h.cleanupJob(ctx)
}

func (h *Hub) HandleSubscription(ctx context.Context, client *websocket.Conn, msg Subscription, id int32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscriptions, ok := h.Clients[client]
	if !ok {
		subscriptions = map[string]struct{}{}
	}

	valid := []string{}
	for _, param := range msg.Params {
		symbol := strings.TrimPrefix(param, "submission@")
		if _, ok := h.Symbols[symbol]; !ok {
			continue
		}
		subscriptions[symbol] = struct{}{}
		valid = append(valid, param)
	}
	h.Clients[client] = subscriptions

	defer func(subscribed []string) {
		if len(valid) == 0 {
			return
		}

		if err := stats.InsertWebsocketSubscriptions(ctx, id, valid); err != nil {
			log.Error().Err(err).Msg("failed to insert websocket subscriptions")
		}
	}(valid)
}

func (h *Hub) handleClientRegistration(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.Register:
			h.addClient(client)
		case client := <-h.Unregister:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *websocket.Conn) {
	h.mu.Lock() // Use write lock for both checking and insertion
	defer h.mu.Unlock()
	if _, ok := h.Clients[client]; ok {
		return
	}
	h.Clients[client] = make(map[string]struct{})
}

func (h *Hub) removeClient(client *websocket.Conn) {
	h.mu.Lock() // Use write lock for both checking and removal
	defer h.mu.Unlock()
	subscriptions, ok := h.Clients[client]
	if !ok {
		return
	}
	delete(h.Clients, client)
	for symbol := range subscriptions {
		delete(subscriptions, symbol)
	}

	err := client.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Warn().Err(err).Msg("failed to write close message")
	}

}

func (h *Hub) initializeBroadcastChannels(collector *collector.Collector) {
	for symbol, stream := range collector.OutgoingStream {
		h.broadcast[symbol] = stream
	}
}

func (h *Hub) broadcastDataForSymbol(ctx context.Context, symbol string) {
	for data := range h.broadcast[symbol] {
		h.castSubmissionData(ctx, data, symbol)
	}
}

func (h *Hub) castSubmissionData(ctx context.Context, data *dalcommon.OutgoingSubmissionData, symbol string) {
	var wg sync.WaitGroup
	var failedMu sync.Mutex
	var failedClients []*websocket.Conn

	h.mu.RLock()
	for client, subscriptions := range h.Clients {
		if _, ok := subscriptions[symbol]; ok {
			wg.Add(1)
			go func(entry *websocket.Conn) {
				defer wg.Done()
				writeCtx, cancel := context.WithTimeout(ctx, WriteTimeout)
				defer cancel()
				if err := wsjson.Write(writeCtx, entry, data); err != nil {
					log.Warn().Err(err).Msg("failed to write message to client")
					failedMu.Lock()
					failedClients = append(failedClients, entry)
					failedMu.Unlock()
				}
			}(client)
		}
	}
	h.mu.RUnlock()
	wg.Wait()

	if len(failedClients) > 0 {
		h.mu.Lock()
		for _, client := range failedClients {
			delete(h.Clients, client)
			client.Close(websocket.StatusGoingAway, "write failed")
		}
		h.mu.Unlock()
	}
}

func (h *Hub) cleanupJob(ctx context.Context) {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.cleanup()
		}
	}
}

func (h *Hub) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client, subscriptions := range h.Clients {
		if len(subscriptions) == 0 {
			delete(h.Clients, client)
			client.Close(websocket.StatusNormalClosure, "")
		}
	}
}
