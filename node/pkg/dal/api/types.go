package api

import (
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"github.com/gofiber/contrib/websocket"
)

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Controller struct {
	Collector  *collector.Collector
	configs    map[string]types.Config
	clients    map[*websocket.Conn]map[string]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  map[string]chan dalcommon.OutgoingSubmissionData
}
