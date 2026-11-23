package dalstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dalcommon "bisonai.com/miko/node/pkg/dal/common"
)

var (
	errMaxConnections = errors.New("too many connections")
	errMaxPerIP       = errors.New("too many connections from this address")
	errNotReady       = errors.New("upstream not ready")
)

// entry is the latest upstream value of a symbol.
type entry struct {
	price   string // value scaled by decimals
	aggTime int64  // unix ms
}

// group holds the clients that requested the same symbol filter, so each
// tick's payload is encoded once per filter.
type group struct {
	key     string
	symbols []string // nil: all symbols
	clients map[*client]struct{}
}

type client struct {
	ip      string
	group   *group
	queue   chan []byte
	cancel  context.CancelFunc
	removed bool // guarded by Hub.mu
}

// Hub keeps the latest value per symbol and the connected clients.
type Hub struct {
	mu sync.Mutex

	known  map[string]struct{} // symbols served by DAL (GET /symbols)
	latest map[string]*entry
	// published is the state as of the last tick; snapshots come from it, so
	// a new client and the following updates agree with older clients
	published map[string]entry
	dirty     map[string]struct{}

	groups map[string]*group
	total  int
	perIP  map[string]int

	connected bool
	connData  bool // data received on the current connection
	lastMsg   time.Time

	maxConns    int
	maxPerIP    int
	clientQueue int
	now         func() time.Time
}

func NewHub(cfg Config) *Hub {
	return &Hub{
		known:       map[string]struct{}{},
		latest:      map[string]*entry{},
		published:   map[string]entry{},
		dirty:       map[string]struct{}{},
		groups:      map[string]*group{},
		perIP:       map[string]int{},
		maxConns:    cfg.MaxConns,
		maxPerIP:    cfg.MaxConnsPerIP,
		clientQueue: cfg.ClientQueue,
		now:         time.Now,
	}
}

// SetSymbols replaces the served symbols and returns those not served before.
// Data of removed symbols is dropped. An empty list is ignored.
func (h *Hub) SetSymbols(symbols []string) (added []string) {
	if len(symbols) == 0 {
		return nil
	}
	next := make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		next[s] = struct{}{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.known {
		if _, ok := next[s]; !ok {
			delete(h.latest, s)
			delete(h.published, s)
			delete(h.dirty, s)
		}
	}
	for s := range next {
		if _, ok := h.known[s]; !ok {
			added = append(added, s)
		}
	}
	h.known = next
	sort.Strings(added)
	return added
}

// Apply records an upstream message. A symbol is marked changed only when
// its scaled value differs; aggregateTime alone does not count.
func (h *Hub) Apply(msg dalcommon.OutgoingSubmissionData) error {
	decimals, err := strconv.Atoi(msg.Decimals)
	if err != nil {
		return fmt.Errorf("%s: invalid decimals %q", msg.Symbol, msg.Decimals)
	}
	aggTime, err := strconv.ParseInt(msg.AggregateTime, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: invalid aggregateTime %q", msg.Symbol, msg.AggregateTime)
	}
	price, err := formatPrice(msg.Value, decimals)
	if err != nil {
		return fmt.Errorf("%s: %w", msg.Symbol, err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastMsg = h.now()
	h.connData = true
	if _, ok := h.known[msg.Symbol]; !ok {
		return nil
	}
	e := h.latest[msg.Symbol]
	if e == nil {
		e = &entry{}
		h.latest[msg.Symbol] = e
	}
	if aggTime < e.aggTime {
		return nil // out of order
	}
	if e.price != price {
		h.dirty[msg.Symbol] = struct{}{}
	}
	e.price, e.aggTime = price, aggTime
	return nil
}

// SetConnected records the upstream connection state.
func (h *Hub) SetConnected(connected bool) {
	h.mu.Lock()
	h.connected = connected
	h.connData = false
	h.mu.Unlock()
	if connected {
		upstreamConnected.Set(1)
	} else {
		upstreamConnected.Set(0)
	}
}

// Healthy reports whether the upstream is connected and sent data within
// staleAfter.
func (h *Hub) Healthy(staleAfter time.Duration) (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch {
	case !h.connected:
		return false, "upstream disconnected"
	case !h.connData:
		return false, "no upstream data yet"
	case h.now().Sub(h.lastMsg) > staleAfter:
		return false, "upstream stale"
	}
	return true, "ok"
}

func (h *Hub) lastMessageAge() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastMsg.IsZero() {
		return -1
	}
	return h.now().Sub(h.lastMsg).Seconds()
}

// ParseFilter validates the comma-separated symbols query. Empty means all
// symbols. The returned key is canonical (sorted, deduplicated).
func (h *Hub) ParseFilter(query string) (key string, symbols []string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.known) == 0 {
		return "", nil, errNotReady
	}
	seen := map[string]struct{}{}
	var unknown []string
	for _, s := range strings.Split(query, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		if _, ok := h.known[s]; !ok {
			unknown = append(unknown, s)
			continue
		}
		symbols = append(symbols, s)
	}
	if len(unknown) > 0 {
		return "", nil, fmt.Errorf("unknown symbols: %s", strings.Join(unknown, ","))
	}
	if len(symbols) == 0 {
		return "", nil, nil
	}
	sort.Strings(symbols)
	return strings.Join(symbols, ","), symbols, nil
}

// Register admits a client for the filter (from ParseFilter) and queues its
// snapshot. cancel is called when the hub drops the client.
func (h *Hub) Register(ip, key string, symbols []string, cancel context.CancelFunc) (*client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.total >= h.maxConns {
		rejectedConnections.WithLabelValues(RejectMaxConnections).Inc()
		return nil, errMaxConnections
	}
	if h.perIP[ip] >= h.maxPerIP {
		rejectedConnections.WithLabelValues(RejectMaxPerIP).Inc()
		return nil, errMaxPerIP
	}
	g := h.groups[key]
	if g == nil {
		g = &group{key: key, symbols: symbols, clients: map[*client]struct{}{}}
		h.groups[key] = g
	}
	c := &client{ip: ip, group: g, queue: make(chan []byte, h.clientQueue), cancel: cancel}
	g.clients[c] = struct{}{}
	h.total++
	h.perIP[ip]++
	activeConnections.Inc()
	connectionsTotal.Inc()

	snap := map[string]entry{}
	if symbols == nil {
		for s, e := range h.published {
			snap[s] = h.snapshotEntry(s, e)
		}
	} else {
		for _, s := range symbols {
			if e, ok := h.published[s]; ok {
				snap[s] = h.snapshotEntry(s, e)
			}
		}
	}
	c.queue <- encodeEvent("snapshot", h.now().UnixMilli(), snap)
	return c, nil
}

// snapshotEntry is the published price with the newest aggregateTime seen for
// it: a price that has not moved keeps a fresh aggregateTime.
func (h *Hub) snapshotEntry(symbol string, pub entry) entry {
	if e := h.latest[symbol]; e != nil && e.price == pub.price {
		return *e
	}
	return pub
}

// Unregister removes a client; safe to call more than once.
func (h *Hub) Unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(c)
}

func (h *Hub) removeLocked(c *client) {
	if c.removed {
		return
	}
	c.removed = true
	delete(c.group.clients, c)
	if len(c.group.clients) == 0 {
		delete(h.groups, c.group.key)
	}
	h.total--
	if h.perIP[c.ip]--; h.perIP[c.ip] <= 0 {
		delete(h.perIP, c.ip)
	}
	activeConnections.Dec()
	c.cancel()
}

// Tick sends each client the requested symbols whose value changed since the
// previous tick. Clients whose queue is full are dropped.
func (h *Hub) Tick() {
	h.mu.Lock()
	defer h.mu.Unlock()
	changes := map[string]entry{}
	for s := range h.dirty {
		e := h.latest[s]
		if e != nil && h.published[s].price != e.price {
			changes[s] = *e
			h.published[s] = *e
		}
	}
	clear(h.dirty)
	changedPerTick.Observe(float64(len(changes)))
	if len(changes) == 0 {
		return
	}

	t := h.now().UnixMilli()
	var all []byte
	for _, g := range h.groups {
		var payload []byte
		if g.symbols == nil {
			if all == nil {
				all = encodeEvent("update", t, changes)
			}
			payload = all
		} else {
			sub := map[string]entry{}
			for _, s := range g.symbols {
				if e, ok := changes[s]; ok {
					sub[s] = e
				}
			}
			if len(sub) == 0 {
				continue
			}
			payload = encodeEvent("update", t, sub)
		}
		for c := range g.clients {
			select {
			case c.queue <- payload:
			default:
				slowClientDrops.Inc()
				h.removeLocked(c)
			}
		}
	}
}

// Run calls Tick every interval until ctx is done.
func (h *Hub) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.Tick()
		}
	}
}

// encodeEvent builds one SSE event:
//
//	event: <name>
//	data: {"t":<unix ms>,"d":{"BTC-USDT":["61410.18",<aggregateTime ms>]}}
func encodeEvent(name string, t int64, items map[string]entry) []byte {
	d := make(map[string][2]any, len(items))
	for s, e := range items {
		d[s] = [2]any{e.price, e.aggTime}
	}
	data, err := json.Marshal(struct {
		T int64             `json:"t"`
		D map[string][2]any `json:"d"`
	}{t, d})
	if err != nil {
		// strings and ints only; cannot fail
		panic(err)
	}
	out := make([]byte, 0, len(name)+len(data)+16)
	out = append(out, "event: "...)
	out = append(out, name...)
	out = append(out, "\ndata: "...)
	out = append(out, data...)
	out = append(out, "\n\n"...)
	return out
}
