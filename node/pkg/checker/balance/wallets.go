package balance

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kaiachain/kaia/common"
)

// EnvWallets lists the watched wallets: comma-separated
// "<address>:<minimum KAIA>[:<role>]" entries. role labels metrics and
// alerts (default "wallet"); keep it generic.
const EnvWallets = "BALANCE_WALLETS"

const defaultRole = "wallet"

var (
	addressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	rolePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
)

// ParseWallets parses BALANCE_WALLETS. Duplicates and malformed entries are
// errors, so a typo fails at start instead of silently dropping a wallet.
func ParseWallets(raw string) ([]Wallet, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New(EnvWallets + " is empty")
	}

	var result []Wallet
	seen := map[common.Address]bool{}
	for i, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Split(entry, ":")
		if len(fields) < 2 || len(fields) > 3 {
			return nil, fmt.Errorf("%s entry %d: want <address>:<minimum>[:<role>]", EnvWallets, i+1)
		}
		address := strings.TrimSpace(fields[0])
		if !addressPattern.MatchString(address) {
			return nil, fmt.Errorf("%s entry %d: invalid address", EnvWallets, i+1)
		}
		minimum, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		if err != nil || minimum < 0 {
			return nil, fmt.Errorf("%s entry %d: invalid minimum %q", EnvWallets, i+1, fields[1])
		}
		role := defaultRole
		if len(fields) == 3 {
			role = strings.TrimSpace(fields[2])
			if !rolePattern.MatchString(role) {
				return nil, fmt.Errorf("%s entry %d: invalid role %q (lowercase letters, digits, '-')", EnvWallets, i+1, role)
			}
		}
		addr := common.HexToAddress(address)
		if seen[addr] {
			return nil, fmt.Errorf("%s entry %d: duplicate address", EnvWallets, i+1)
		}
		seen[addr] = true
		result = append(result, Wallet{
			Tag:     role,
			Address: addr,
			Minimum: minimum,
		})
	}
	if len(result) == 0 {
		return nil, errors.New(EnvWallets + " has no entries")
	}
	return result, nil
}
