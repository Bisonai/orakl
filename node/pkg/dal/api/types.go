package api

import (
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
)

const (
	MAX_CONNECTIONS = 10
	CleanupInterval = time.Hour
)

type Config = types.Config

type Subscription struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Hub struct {
	configs    map[string]Config
	clients    map[*ThreadSafeClient]map[string]any
	register   chan *ThreadSafeClient
	unregister chan *ThreadSafeClient
	broadcast  map[string]chan *dalcommon.OutgoingSubmissionData
	connPerIP  map[string][]*ThreadSafeClient
	mu         sync.RWMutex
}

type BulkResponse struct {
	Symbols        []string `json:"symbols"`
	Values         []string `json:"values"`
	AggregateTimes []string `json:"aggregateTimes"`
	Proofs         []string `json:"proofs"`
	FeedHashes     []string `json:"feedHashes"`
	Decimals       []string `json:"decimals"`
}
