package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetConfigUrl verifies the config-bundle URL resolves to the chain-renamed
// `_feeds.json` form for the mainnet/kairos chain values, and that an unset
// CHAIN defaults to kairos.
func TestGetConfigUrl(t *testing.T) {
	cases := map[string]string{
		"mainnet": "https://config.orakl.network/mainnet_feeds.json",
		"kairos":  "https://config.orakl.network/kairos_feeds.json",
	}
	for chain, want := range cases {
		t.Setenv("CHAIN", chain)
		assert.Equal(t, want, getConfigUrl(), "getConfigUrl() with CHAIN=%s", chain)
	}

	// CHAIN unset defaults to kairos.
	t.Setenv("CHAIN", "")
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", getConfigUrl())
}
