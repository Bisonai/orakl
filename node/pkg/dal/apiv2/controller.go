package apiv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
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

	l, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		return err
	}

	wsServer := NewServer(config.Collector, config.KeyCache, config.Hub, config.StatsApp)
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

func NewServer(collector *collector.Collector, keyCache *keycache.KeyCache, hub *hub.Hub, statsApp *stats.StatsApp) *ServerV2 {
	s := &ServerV2{
		collector: collector,
		keyCache:  keyCache,
		hub:       hub,
	}
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/", s.HealthCheckHandler)
	serveMux.HandleFunc("/ws", s.WSHandler)

	serveMux.HandleFunc("GET /symbols", s.SymbolsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/all", s.AllLatestFeedsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/transpose/all", s.AllLatestFeedsTransposedHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/transpose/{symbols}", s.TransposedLatestFeedsHandler)
	serveMux.HandleFunc("GET /latest-data-feeds/{symbols}", s.LatestFeedsHandler)

	// Apply the RequestLoggerMiddleware to the ServeMux
	loggedMux := statsApp.RequestLoggerMiddleware(serveMux)

	s.handler = loggedMux

	return s
}

func (s *ServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-API-Key")

	if r.RequestURI != "/" && !s.checkAPIKey(r.Context(), key) {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte("Unauthorized"))
		if err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	s.handler.ServeHTTP(w, r)
}

func (s *ServerV2) checkAPIKey(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}

	if s.keyCache.Get(key) {
		return true
	}

	if keycache.ValidateApiKeyFromDB(ctx, key) {
		s.keyCache.Set(key)
		return true
	}

	return false
}

func (s *ServerV2) WSHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept websocket connection")
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	s.hub.Register <- c

	key := r.Header.Get("X-API-Key")
	id, err := stats.InsertWebsocketConnection(r.Context(), key)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert websocket connection")
		return
	}
	log.Info().Int32("id", id).Msg("inserted websocket connection")

	defer func() {
		s.hub.Unregister <- c
		err = stats.UpdateWebsocketConnection(r.Context(), id)
		if err != nil {
			log.Error().Err(err).Msg("failed to update websocket connection")
			return
		}
		log.Info().Int32("id", id).Msg("updated websocket connection")
	}()

	for {
		var msg hub.Subscription
		if err = wsjson.Read(r.Context(), c, &msg); err != nil {
			log.Error().Err(err).Msg("failed to read message")
			return
		}

		if msg.Method == "SUBSCRIBE" {
			s.hub.HandleSubscription(r.Context(), c, msg, id)
		}
	}
}

func (s *ServerV2) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Orakl Node DAL API"))
	if err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

func (s *ServerV2) SymbolsHandler(w http.ResponseWriter, r *http.Request) {
	result := make([]string, 0, len(s.hub.Configs))
	for key := range s.hub.Configs {
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
