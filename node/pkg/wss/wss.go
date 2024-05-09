package wss

import (
	"context"
	"fmt"
	"sync"

	"nhooyr.io/websocket"
)

type Connections struct {
	m    map[string]*websocket.Conn
	lock sync.RWMutex
}

var (
	connections *Connections
)

func init() {
	connections = &Connections{
		m: make(map[string]*websocket.Conn),
	}
}

func (c *Connections) Update(key string, conn *websocket.Conn) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.m[key] != nil {
		err := c.m[key].Close(websocket.StatusNormalClosure, "")
		if err != nil {
			return err
		}
	}
	c.m[key] = conn
	return nil
}

func (c *Connections) Remove(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.m[key] != nil {
		err := c.m[key].Close(websocket.StatusNormalClosure, "")
		if err != nil {
			return err
		}
	}
	delete(c.m, key)
	return nil
}

func (c *Connections) Get(key string) (*websocket.Conn, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	conn, ok := c.m[key]
	if !ok {
		return nil, fmt.Errorf("connection not found")
	}
	return conn, nil
}

func getConnections() *Connections {
	return connections
}

func UpdateConnection(ctx context.Context, key string, conn *websocket.Conn) error {
	connections := getConnections()
	return connections.Update(key, conn)
}

func GetConnection(key string) (*websocket.Conn, error) {
	connections := getConnections()
	return connections.Get(key)
}

func RemoveConnection(key string) error {
	connections := getConnections()
	return connections.Remove(key)
}
