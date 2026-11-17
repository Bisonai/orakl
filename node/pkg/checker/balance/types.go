package balance

import (
	"time"

	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
)

var BalanceCheckInterval time.Duration

var kaiaClient *client.Client
var wallets []Wallet

type Wallet struct {
	// Tag is the wallet's role (BALANCE_WALLETS), used in metrics and alerts.
	Tag     string
	Address common.Address `db:"address" json:"address"`
	Balance float64        `db:"balance" json:"balance"`
	Minimum float64
}
