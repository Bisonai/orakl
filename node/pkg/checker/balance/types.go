package balance

import (
	"time"

	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
)

const (
	BalanceHistoryTTL        = 60 * time.Minute
	MinimalIncreaseThreshold = 0.5
	ChunkSize                = 10
)

var BalanceCheckInterval time.Duration
var BalanceAlarmInterval time.Duration

var kaiaClient *client.Client
var wallets []Wallet

type Wallet struct {
	// Tag is the wallet's role (BALANCE_WALLETS), used in metrics and alerts.
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
