package api

import (
	"sync"

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
	Collector *collector.Collector

	configs    map[string]types.Config
	clients    map[*websocket.Conn]map[string]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  map[string]chan dalcommon.OutgoingSubmissionData

	mu sync.RWMutex
}

type BulkResponse struct {
	Symbols        []string   `json:"symbols"`
	Values         []string   `json:"values"`
	AggregateTimes []string   `json:"aggregateTimes"`
	Proofs         [][]byte   `json:"proofs"`
	FeedHashes     [][32]byte `json:"feedHashes"`
	Decimals       []string   `json:"decimals"`
}
