package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDelegatorAlarmGate verifies the mainnet delegator-threshold gate fires for
// the mainnet chain both before and after infra flips CHAIN from cypress to
// mainnet, and stays at the default for the testnet chain (baobab/kairos).
func TestDelegatorAlarmGate(t *testing.T) {
	// Isolate from a DELEGATOR_ALARM_AMOUNT override so the test exercises the
	// chain-specific threshold, not an env-provided value.
	t.Setenv("DELEGATOR_ALARM_AMOUNT", "")

	cases := map[string]float64{
		"cypress": 50000,
		"mainnet": 50000,
		"baobab":  10000,
		"kairos":  10000,
	}
	for chain, want := range cases {
		t.Setenv("CHAIN", chain)
		loadEnvs()
		assert.Equal(t, want, DelegatorAlarmAmount, "DelegatorAlarmAmount with CHAIN=%s", chain)
	}
}
