package api

import (
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

type ThreadSafeClient struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func NewThreadSafeClient(conn *websocket.Conn) *ThreadSafeClient {
	return &ThreadSafeClient{
		Conn: conn,
	}
}

func (c *ThreadSafeClient) WriteJSON(data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.Conn.WriteJSON(data); err != nil {
		log.Error().Err(err).Msg("failed to write json msg")
		return err
	}
	return nil
}

// even though readjson is not thread safe, it is expected not to be called concurrently
// since the only place it is called is from `HandleWebsocket` inner for loop
func (c *ThreadSafeClient) ReadJSON(data any) error {
	if err := c.Conn.ReadJSON(&data); err != nil {
		return err
	}
	return nil
}

func (c *ThreadSafeClient) Close() error {
	return c.Conn.Close()
}

func (c *ThreadSafeClient) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return c.Conn.WriteControl(messageType, data, deadline)
}
