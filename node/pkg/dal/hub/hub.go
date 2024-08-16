package hub

import (
	"context"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Config = types.Config

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Hub struct {
	Configs    map[string]Config
	Clients    map[*websocket.Conn]map[string]any
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	broadcast  map[string]chan *dalcommon.OutgoingSubmissionData
	mu         sync.RWMutex
}

const (
	MAX_CONNECTIONS = 10
	CleanupInterval = time.Hour
)

func HubSetup(ctx context.Context, configs []Config) *Hub {
	configMap := make(map[string]Config)
	for _, config := range configs {
		configMap[config.Name] = config
	}

	hub := NewHub(configMap)
	return hub
}

func NewHub(configs map[string]Config) *Hub {
	return &Hub{
		Configs:    configs,
		Clients:    make(map[*websocket.Conn]map[string]any),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
		broadcast:  make(map[string]chan *dalcommon.OutgoingSubmissionData),
	}
}

func (h *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go h.handleClientRegistration()

	h.initializeBroadcastChannels(collector)

	for symbol := range h.Configs {
		go h.broadcastDataForSymbol(ctx, symbol)
	}

	go h.cleanupJob(ctx)
}

func (h *Hub) HandleSubscription(ctx context.Context, client *websocket.Conn, msg Subscription, id int32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscriptions, ok := h.Clients[client]
	if !ok {
		subscriptions = map[string]any{}
	}

	valid := []string{}
	for _, param := range msg.Params {
		symbol := strings.TrimPrefix(param, "submission@")
		if _, ok := h.Configs[symbol]; !ok {
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

func (h *Hub) handleClientRegistration() {
	for {
		select {
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
	h.Clients[client] = make(map[string]any)
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
	for configId, stream := range collector.OutgoingStream {
		symbol := h.configIdToSymbol(configId)
		if symbol == "" {
			continue
		}

		h.broadcast[symbol] = stream
	}
}

func (h *Hub) configIdToSymbol(id int32) string {
	for symbol, config := range h.Configs {
		if config.ID == id {
			return symbol
		}
	}
	return ""
}

func (h *Hub) broadcastDataForSymbol(ctx context.Context, symbol string) {
	for data := range h.broadcast[symbol] {
		go h.castSubmissionData(ctx, data, &symbol)
	}
}

func (h *Hub) castSubmissionData(ctx context.Context, data *dalcommon.OutgoingSubmissionData, symbol *string) {
	var wg sync.WaitGroup

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client, subscriptions := range h.Clients {
		if _, ok := subscriptions[*symbol]; ok {
			wg.Add(1)
			go func(entry *websocket.Conn) {
				defer wg.Done()
				wsjson.Write(ctx, entry, data)
			}(client)
		}
	}
	wg.Wait()
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

	newClients := make(map[*websocket.Conn]map[string]any, len(h.Clients))
	for client, subscriptions := range h.Clients {
		if len(subscriptions) > 0 {
			newClients[client] = subscriptions
		} else {
			h.Unregister <- client
		}
	}
	h.Clients = newClients
}
