package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetConfigUrl verifies the migrated config-bundle URL resolves to the new
// chain-renamed `_feeds.json` form both before and after infra flips CHAIN from
// the old value (cypress/baobab) to the new one (mainnet/kairos).
func TestGetConfigUrl(t *testing.T) {
	cases := map[string]string{
		"cypress": "https://config.orakl.network/mainnet_feeds.json",
		"mainnet": "https://config.orakl.network/mainnet_feeds.json",
		"baobab":  "https://config.orakl.network/kairos_feeds.json",
		"kairos":  "https://config.orakl.network/kairos_feeds.json",
	}
	for chain, want := range cases {
		t.Setenv("CHAIN", chain)
		assert.Equal(t, want, getConfigUrl(), "getConfigUrl() with CHAIN=%s", chain)
	}

	// CHAIN unset defaults to baobab -> kairos.
	t.Setenv("CHAIN", "")
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", getConfigUrl())
}
