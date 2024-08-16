package wsserver

import (
	"context"
	"net/http"

	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WsServer struct {
	hub      *hub.Hub
	keyCache *keycache.KeyCache
	serveMux http.ServeMux
}

func NewWSServer(keyCache *keycache.KeyCache, hub *hub.Hub) *WsServer {
	s := &WsServer{
		keyCache: keyCache,
		hub:      hub,
	}
	s.serveMux.HandleFunc("/ws", s.Handler)
	return s
}

func (s *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-API-Key")
	if !s.checkAPIKey(r.Context(), key) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.serveMux.ServeHTTP(w, r)
}

func (s *WsServer) Handler(w http.ResponseWriter, r *http.Request) {
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

func (s *WsServer) checkAPIKey(ctx context.Context, key string) bool {
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
