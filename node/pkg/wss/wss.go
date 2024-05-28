package wss

import (
	"context"
	"fmt"
	"sync"
)

type Connections struct {
	m    map[string]*WebsocketHelper
	lock sync.RWMutex
}

var (
	connections *Connections
)

func init() {
	connections = &Connections{
		m: make(map[string]*WebsocketHelper),
	}
}

func (c *Connections) Update(key string, conn *WebsocketHelper) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.m[key] != nil {
		err := c.m[key].Close()
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

	if c.m[key] != nil && c.m[key].Conn != nil {
		err := c.m[key].Close()
		if err != nil {
			return err
		}
	}
	delete(c.m, key)
	return nil
}

func (c *Connections) Get(key string) (*WebsocketHelper, error) {
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

func UpdateConnection(ctx context.Context, key string, conn *WebsocketHelper) error {
	connections := getConnections()
	return connections.Update(key, conn)
}

func GetConnection(key string) (*WebsocketHelper, error) {
	connections := getConnections()
	return connections.Get(key)
}

func RemoveConnection(key string) error {
	connections := getConnections()
	return connections.Remove(key)
}
