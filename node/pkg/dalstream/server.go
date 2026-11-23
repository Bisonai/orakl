package dalstream

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// Server serves the public endpoints: GET /stream and GET /health.
type Server struct {
	hub *Hub
	cfg Config
	mux *http.ServeMux
}

func NewServer(hub *Hub, cfg Config) *Server {
	s := &Server{hub: hub, cfg: cfg, mux: http.NewServeMux()}
	s.mux.HandleFunc("/stream", s.handleStream)
	s.mux.HandleFunc("/health", s.handleHealth)
	return s
}

// ServeHTTP applies the method and CORS rules, then routes.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		if _, ok := s.cfg.AllowedOrigins[origin]; !ok {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Add("Vary", "Origin")
	}
	if r.Method == http.MethodOptions {
		h := w.Header()
		h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
			// EventSource sends no custom headers; allow the usual ones
			h.Set("Access-Control-Allow-Headers", "Cache-Control, Last-Event-ID")
		}
		h.Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	ok, reason := s.hub.Healthy(s.cfg.StaleAfter)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_, _ = w.Write([]byte(reason + "\n"))
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	key, symbols, err := s.hub.ParseFilter(r.URL.Query().Get("symbols"))
	if errors.Is(err, errNotReady) {
		w.Header().Set("Retry-After", strconv.Itoa(RetryMillis/1000))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ip := limitKey(clientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), s.cfg.TrustedProxies))
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c, err := s.hub.Register(ip, key, symbols, cancel)
	switch {
	case errors.Is(err, errMaxConnections):
		w.Header().Set("Retry-After", "10")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	case errors.Is(err, errMaxPerIP):
		w.Header().Set("Retry-After", "10")
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer s.hub.Unregister(c)

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	write := func(b []byte) bool {
		if err := rc.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		n, err := w.Write(b)
		bytesSent.Add(float64(n))
		if err == nil {
			err = rc.Flush()
		}
		if err != nil {
			log.Debug().Err(err).Str("ip", ip).Msg("stream write failed")
			return false
		}
		return true
	}

	if !write([]byte("retry: " + strconv.Itoa(RetryMillis) + "\n\n")) {
		return
	}
	ping := time.NewTicker(s.cfg.PingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-c.queue:
			if !write(b) {
				return
			}
		case <-ping.C:
			if !write([]byte(": ping\n\n")) {
				return
			}
		}
	}
}
