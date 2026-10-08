package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigUrls verifies the bundle URLs resolve to the canonical
// chain-renamed form for the mainnet/kairos chain values, and that the
// untouched peg.por URL keeps the raw chain value.
func TestConfigUrls(t *testing.T) {
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", loadMikoConfigUrl("mainnet"))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", loadMikoConfigUrl("kairos"))
	assert.Equal(t, "https://config.orakl.network/mainnet_mag7.json", loadPorInfoUrl("mainnet"))
	assert.Equal(t, "https://config.orakl.network/kairos_mag7.json", loadPorInfoUrl("kairos"))

	// loadPegPorConfigUrl is out of scope (#2576) and keeps the raw chain value.
	assert.Equal(t, "https://config.orakl.network/aggregator/mainnet/peg.por.json", loadPegPorConfigUrl("mainnet"))
}
