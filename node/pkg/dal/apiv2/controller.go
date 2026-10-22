package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"bisonai.com/miko/node/pkg/dal/collector"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func Start(ctx context.Context, opts ...ServerV2Option) error {
	config := &ServerV2Config{
		Port: "8090",
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.Port == "" {
		return errorsentinel.ErrDalPortNotFound
	}

	if config.Collector == nil {
		return errorsentinel.ErrDalCollectorNotFound
	}

	if config.Hub == nil {
		return errorsentinel.ErrDalHubNotFound
	}

	if config.KeyCache == nil {
		return errorsentinel.ErrDalKeyCacheNotFound
	}

	if config.KeyStore == nil {
		return errorsentinel.ErrDalKeyStoreNotFound
	}

	l, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		return err
	}

	wsServer := NewServer(config.Collector, config.KeyCache, config.KeyStore, config.Hub)
	httpServer := &http.Server{
		Handler: wsServer,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	err = httpServer.Serve(l)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func NewServer(collector *collector.Collector, keyCache *keycache.KeyCache, keyStore *keystore.Store, hub *hub.Hub) *ServerV2 {
	s := &ServerV2{
		collector: collector,
		keyCache:  keyCache,
		keyStore:  keyStore,
		hub:       hub,
	}
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/ws", s.WSHandler)

	serveMux.HandleFunc("GET /symbols", s.SymbolsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/all", s.AllLatestFeedsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/transpose/all", s.AllLatestFeedsTransposedHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/transpose/{symbols}", s.TransposedLatestFeedsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/{symbols}", s.LatestFeedsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds-unstrict/{symbols}", s.LatestFeedsHandlerUnstrict)

	serveMux.Handle("GET /metrics", promhttp.Handler())
	serveMux.HandleFunc("/", s.HealthCheckHandler)

	s.mux = serveMux

	return s
}

func (s *ServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/metrics" {
		s.mux.ServeHTTP(w, r)
		return
	}

	isWS := r.URL.Path == "/ws"

	label, ok, err := s.checkAPIKey(r.Context(), r.Header.Get("X-API-Key"))
	if err != nil || !ok {
		status, body := http.StatusUnauthorized, "Unauthorized"
		if err != nil {
			status, body = http.StatusServiceUnavailable, "Service Unavailable"
		}
		if !isWS {
			_, pattern := s.mux.Handler(r)
			observeREST(UnknownKeyLabel, endpointLabel(pattern), status)
		}
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	r = r.WithContext(withKeyLabel(r.Context(), label))

	// websocket traffic is tracked by the dal_websocket_* metrics
	if isWS {
		s.mux.ServeHTTP(w, r)
		return
	}

	rec := newStatusRecorder(w)
	s.mux.ServeHTTP(rec, r)
	// ServeMux sets r.Pattern on the request it routes
	observeREST(label, endpointLabel(r.Pattern), rec.status)
}

// checkAPIKey validates key against the cache, then the key store. It returns
// the key's metric label (its description) and whether the key is valid; err
// is set when the key store could not be queried.
func (s *ServerV2) checkAPIKey(ctx context.Context, key string) (string, bool, error) {
	if key == "" {
		return "", false, nil
	}

	if description, ok := s.keyCache.Get(key); ok {
		return keyLabel(description), true, nil
	}

	description, ok, err := s.keyStore.Lookup(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Error validating API key")
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	s.keyCache.Set(key, description)
	return keyLabel(description), true, nil
}

func (s *ServerV2) WSHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept websocket connection")
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	label := keyLabelFrom(r.Context())

	s.hub.Register <- c
	wsConnectionsTotal.WithLabelValues(label).Inc()
	wsActiveConnections.WithLabelValues(label).Inc()

	defer func() {
		s.hub.Unregister <- c
		wsActiveConnections.WithLabelValues(label).Dec()
	}()

	for {
		var msg hub.Subscription
		if err = wsjson.Read(r.Context(), c, &msg); err != nil {
			log.Error().Err(err).Msg("failed to read message")
			return
		}

		if msg.Method == "SUBSCRIBE" {
			valid := s.hub.HandleSubscription(c, msg)
			wsSubscriptionsTotal.WithLabelValues(label).Add(float64(valid))
		}
	}
}

func (s *ServerV2) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Miko Node DAL API"))
	if err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

func (s *ServerV2) SymbolsHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]string, 0, len(s.hub.Symbols))
	for key := range s.hub.Symbols {
		result = append(result, key)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *ServerV2) AllLatestFeedsHandler(w http.ResponseWriter, r *http.Request) {
	result := s.collector.GetAllLatestData()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *ServerV2) AllLatestFeedsTransposedHandler(w http.ResponseWriter, r *http.Request) {
	result := s.collector.GetAllLatestData()
	bulk := BulkResponse{
		Symbols:        make([]string, len(result)),
		Values:         make([]string, len(result)),
		AggregateTimes: make([]string, len(result)),
		Proofs:         make([]string, len(result)),
		FeedHashes:     make([]string, len(result)),
		Decimals:       make([]string, len(result)),
	}
	for i, data := range result {
		bulk.Symbols[i] = data.Symbol
		bulk.Values[i] = data.Value
		bulk.AggregateTimes[i] = data.AggregateTime
		bulk.Proofs[i] = data.Proof
		bulk.FeedHashes[i] = data.FeedHash
		bulk.Decimals[i] = data.Decimals
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(bulk)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *ServerV2) TransposedLatestFeedsHandler(w http.ResponseWriter, r *http.Request) {
	symbolsStr := r.PathValue("symbols")

	if symbolsStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("invalid symbol: empty symbol"))
		if err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	symbolsStr = strings.ReplaceAll(symbolsStr, " ", "")

	symbols := strings.Split(symbolsStr, ",")
	bulk := BulkResponse{}
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		if !strings.Contains(symbol, "-") {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(fmt.Sprintf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}

		result, err := s.collector.GetLatestData(symbol)
		if err != nil {
			log.Error().Err(err).Msg("failed to get latest data")
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		bulk.Symbols = append(bulk.Symbols, result.Symbol)
		bulk.Values = append(bulk.Values, result.Value)
		bulk.AggregateTimes = append(bulk.AggregateTimes, result.AggregateTime)
		bulk.Proofs = append(bulk.Proofs, result.Proof)
		bulk.FeedHashes = append(bulk.FeedHashes, result.FeedHash)
		bulk.Decimals = append(bulk.Decimals, result.Decimals)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(bulk)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *ServerV2) LatestFeedsHandler(w http.ResponseWriter, r *http.Request) {
	symbolsStr := r.PathValue("symbols")
	if symbolsStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("invalid symbol: empty symbol"))
		if err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	symbolsStr = strings.ReplaceAll(symbolsStr, " ", "")

	symbols := strings.Split(symbolsStr, ",")
	results := make([]*dalcommon.OutgoingSubmissionData, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		if !strings.Contains(symbol, "-") {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(fmt.Sprintf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}

		result, err := s.collector.GetLatestData(symbol)
		if err != nil {
			log.Error().Err(err).Msg("failed to get latest data")
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(results)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *ServerV2) LatestFeedsHandlerUnstrict(w http.ResponseWriter, r *http.Request) {
	symbolsStr := r.PathValue("symbols")
	if symbolsStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("invalid symbol: empty symbol"))
		if err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	symbolsStr = strings.ReplaceAll(symbolsStr, " ", "")

	symbols := strings.Split(symbolsStr, ",")
	results := make([]*dalcommon.OutgoingSubmissionData, 0, len(symbols))
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		if !strings.Contains(symbol, "-") {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(fmt.Sprintf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}

		result, err := s.collector.GetLatestData(symbol)
		if err != nil {
			continue
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(results)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}
