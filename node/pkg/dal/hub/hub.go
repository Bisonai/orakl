package hub

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/collector"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
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
	// ctx is set by Start; symbols added after it get their broadcaster at once.
	ctx context.Context
	mu  sync.RWMutex
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
	}
}

// Start starts the hub's goroutines and returns.
func (h *Hub) Start(ctx context.Context, collector *collector.Collector) {
	go h.handleClientRegistration(ctx)

	h.mu.Lock()
	h.ctx = ctx
	streams := collector.Streams()
	for symbol := range h.Symbols {
		if stream, ok := streams[symbol]; ok {
			go h.broadcastDataForSymbol(ctx, symbol, stream)
		}
	}
	h.mu.Unlock()

	go h.cleanupJob(ctx)
}

// SymbolList returns the subscribable symbols.
func (h *Hub) SymbolList() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.Symbols))
	for symbol := range h.Symbols {
		out = append(out, symbol)
	}
	return out
}

// AddSymbols makes symbols subscribable and broadcasts their streams (a
// config reload added them).
func (h *Hub) AddSymbols(streams map[string]chan *dalcommon.OutgoingSubmissionData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for symbol, stream := range streams {
		if _, ok := h.Symbols[symbol]; ok {
			continue
		}
		h.Symbols[symbol] = struct{}{}
		if h.ctx != nil {
			go h.broadcastDataForSymbol(h.ctx, symbol, stream)
		}
	}
}

// RemoveSymbols stops serving symbols (a config reload removed them): they are
// no longer subscribable and are dropped from client subscriptions. Clients
// are not notified; they just stop receiving the symbol. The broadcaster ends
// when the collector closes the stream.
func (h *Hub) RemoveSymbols(symbols []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, symbol := range symbols {
		delete(h.Symbols, symbol)
		for _, subscriptions := range h.Clients {
			delete(subscriptions, symbol)
		}
	}
}

// HandleSubscription subscribes client to the known symbols in msg and returns
// the number of new subscriptions (known symbols not already subscribed).
func (h *Hub) HandleSubscription(client *websocket.Conn, msg Subscription) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscriptions, ok := h.Clients[client]
	if !ok {
		subscriptions = map[string]struct{}{}
	}

	valid := 0
	for _, param := range msg.Params {
		symbol := strings.TrimPrefix(param, "submission@")
		if _, ok := h.Symbols[symbol]; !ok {
			continue
		}
		if _, dup := subscriptions[symbol]; dup {
			continue
		}
		subscriptions[symbol] = struct{}{}
		valid++
	}
	h.Clients[client] = subscriptions

	return valid
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
	subscriptions, ok := h.Clients[client]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.Clients, client)
	for symbol := range subscriptions {
		delete(subscriptions, symbol)
	}
	h.mu.Unlock()

	// Close after unlock: the close handshake can block for seconds.
	closeAsync([]*websocket.Conn{client}, websocket.StatusNormalClosure, "")
}

// closeAsync closes clients in the background, outside h.mu: Close waits up to
// ~5 s each way for the peer's close frame, which must not stall the hub.
// Closing an already closed conn is a no-op in nhooyr websocket, and Close is
// safe to call concurrently with an in-flight Write.
func closeAsync(clients []*websocket.Conn, code websocket.StatusCode, reason string) {
	for _, client := range clients {
		go func(c *websocket.Conn) {
			if err := c.Close(code, reason); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Warn().Err(err).Msg("failed to write close message")
			}
		}(client)
	}
}

func (h *Hub) broadcastDataForSymbol(ctx context.Context, symbol string, stream chan *dalcommon.OutgoingSubmissionData) {
	for data := range stream {
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
		}
		h.mu.Unlock()
		closeAsync(failedClients, websocket.StatusGoingAway, "write failed")
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
	var idle []*websocket.Conn
	h.mu.Lock()
	for client, subscriptions := range h.Clients {
		if len(subscriptions) == 0 {
			delete(h.Clients, client)
			idle = append(idle, client)
		}
	}
	h.mu.Unlock()

	closeAsync(idle, websocket.StatusNormalClosure, "")
}
