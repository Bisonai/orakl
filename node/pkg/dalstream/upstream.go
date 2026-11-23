package dalstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

const (
	upstreamRequestTimeout = 10 * time.Second
	// subscribeChunk bounds a SUBSCRIBE message well below DAL's read limit.
	subscribeChunk = 100
	// readLimit for one DAL message (a single submission is ~0.5 KB).
	readLimit = 1 << 20
)

// Upstream keeps one WebSocket to DAL subscribed to every symbol and feeds
// its messages into the hub.
type Upstream struct {
	cfg  Config
	hub  *Hub
	http *http.Client
}

func NewUpstream(cfg Config, hub *Hub) *Upstream {
	return &Upstream{cfg: cfg, hub: hub, http: &http.Client{Timeout: upstreamRequestTimeout}}
}

// Run connects and reconnects with backoff and jitter until ctx is done.
func (u *Upstream) Run(ctx context.Context) {
	backoff := u.cfg.MinBackoff
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			upstreamReconnects.Inc()
		}
		received, err := u.runOnce(ctx)
		u.hub.SetConnected(false)
		if ctx.Err() != nil {
			return
		}
		if received {
			backoff = u.cfg.MinBackoff
		}
		// full jitter in [backoff/2, backoff)
		wait := backoff/2 + time.Duration(rand.Int64N(int64(backoff/2)+1))
		log.Warn().Err(err).Dur("retryIn", wait).Msg("DAL upstream disconnected")
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		backoff = min(backoff*2, u.cfg.MaxBackoff)
	}
}

// runOnce runs one connection; received reports whether any data arrived.
func (u *Upstream) runOnce(ctx context.Context) (received bool, err error) {
	symbols, err := u.fetchSymbols(ctx)
	if err != nil {
		return false, err
	}
	if len(symbols) == 0 {
		return false, errors.New("DAL returned no symbols")
	}
	u.hub.SetSymbols(symbols)

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	dialCtx, dialCancel := context.WithTimeout(connCtx, upstreamRequestTimeout)
	conn, _, err := websocket.Dial(dialCtx, wsURL(u.cfg.DALURL), &websocket.DialOptions{
		HTTPHeader: http.Header{"X-API-Key": []string{u.cfg.APIKey}},
	})
	dialCancel()
	if err != nil {
		return false, fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(readLimit)

	if err := u.subscribe(connCtx, conn, symbols); err != nil {
		return false, err
	}
	u.hub.SetConnected(true)
	log.Info().Int("symbols", len(symbols)).Msg("DAL upstream connected")

	go u.refreshSymbols(connCtx, cancel, conn)

	for {
		readCtx, readCancel := context.WithTimeout(connCtx, u.cfg.StaleAfter)
		_, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			if connCtx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("no DAL message for %s", u.cfg.StaleAfter)
			}
			return received, err
		}
		received = true
		var msg dalcommon.OutgoingSubmissionData
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Warn().Err(err).Msg("invalid DAL message")
			continue
		}
		if err := u.hub.Apply(msg); err != nil {
			log.Warn().Err(err).Msg("invalid DAL message")
		}
	}
}

// refreshSymbols re-fetches /symbols and re-subscribes to all of them: DAL
// drops subscriptions of a removed symbol, so one removed and re-added between
// two refreshes would otherwise stay silent (DAL ignores duplicates). A failed
// subscribe closes the connection, so it is rebuilt.
func (u *Upstream) refreshSymbols(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	t := time.NewTicker(u.cfg.SymbolsRefresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		symbols, err := u.fetchSymbols(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to refresh DAL symbols")
			continue
		}
		if len(symbols) == 0 {
			continue
		}
		if added := u.hub.SetSymbols(symbols); len(added) > 0 {
			log.Info().Strs("added", added).Msg("new DAL symbols")
		}
		if err := u.subscribe(ctx, conn, symbols); err != nil {
			log.Warn().Err(err).Msg("failed to subscribe to DAL symbols")
			cancel()
			return
		}
	}
}

func (u *Upstream) subscribe(ctx context.Context, conn *websocket.Conn, symbols []string) error {
	for i := 0; i < len(symbols); i += subscribeChunk {
		chunk := symbols[i:min(i+subscribeChunk, len(symbols))]
		params := make([]string, len(chunk))
		for j, s := range chunk {
			params[j] = "submission@" + s
		}
		b, err := json.Marshal(map[string]any{"method": "SUBSCRIBE", "params": params})
		if err != nil {
			return err
		}
		wctx, wcancel := context.WithTimeout(ctx, upstreamRequestTimeout)
		err = conn.Write(wctx, websocket.MessageText, b)
		wcancel()
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}
	}
	return nil
}

func (u *Upstream) fetchSymbols(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.cfg.DALURL+"/symbols", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", u.cfg.APIKey)
	resp, err := u.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /symbols: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("GET /symbols: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var symbols []string
	if err := json.NewDecoder(io.LimitReader(resp.Body, readLimit)).Decode(&symbols); err != nil {
		return nil, fmt.Errorf("GET /symbols: %w", err)
	}
	return symbols, nil
}
