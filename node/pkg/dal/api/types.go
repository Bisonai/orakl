package api

import (
	"sync"

	"bisonai.com/orakl/node/pkg/common/types"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"github.com/gofiber/contrib/websocket"
)

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Hub struct {
	configs    map[string]types.Config
	clients    map[*websocket.Conn]map[string]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  map[string]chan dalcommon.OutgoingSubmissionData

	mu sync.RWMutex
}

type BulkResponse struct {
	Symbols        []string `json:"symbols"`
	Values         []string `json:"values"`
	AggregateTimes []string `json:"aggregateTimes"`
	Proofs         []string `json:"proofs"`
	FeedHashes     []string `json:"feedHashes"`
	Decimals       []string `json:"decimals"`
}
