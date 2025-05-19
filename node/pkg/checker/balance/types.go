package balance

import (
	"time"

	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
)

const (
	mikoApiEndpoint          = "/reporter"
	mikoDelegatorEndpoint    = "/sign/feePayer"
	porEndpoint              = "/address"
	DefaultRRMinimum         = 1
	BalanceHistoryTTL        = 60 * time.Minute
	MinimalIncreaseThreshold = 0.5
	ChunkSize                = 10
)

var SubmitterAlarmAmount float64
var DelegatorAlarmAmount float64
var BalanceCheckInterval time.Duration
var BalanceAlarmInterval time.Duration

var kaiaClient *client.Client
var wallets []Wallet

type Urls struct {
	JsonRpcUrl       string
	MikoApiUrl       string
	MikoNodeAdminUrl string
	MikoDelegatorUrl string
	PorUrl           string
}

type Wallet struct {
	Tag     string
	Address common.Address `db:"address" json:"address"`
	Balance float64        `db:"balance" json:"balance"`
	Minimum float64

	BalanceHistory    []BalanceHistoryEntry
	CurrentDrainRate  float64
	PreviousDrainRate float64
}

type BalanceHistoryEntry struct {
	Timestamp time.Time
	Balance   float64
}

func (b *BalanceHistoryEntry) IsRecent(cutoff time.Time) bool {
	return b.Timestamp.After(cutoff)
}
