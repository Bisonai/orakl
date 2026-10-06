package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigUrls verifies the migrated bundle URLs resolve to the new canonical
// chain-renamed form while infra still injects the old chain value, and that
// the untouched peg.por URL keeps the raw chain value.
func TestConfigUrls(t *testing.T) {
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", loadMikoConfigUrl("cypress"))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", loadMikoConfigUrl("baobab"))
	assert.Equal(t, "https://config.orakl.network/mainnet_mag7.json", loadPorInfoUrl("cypress"))
	assert.Equal(t, "https://config.orakl.network/kairos_mag7.json", loadPorInfoUrl("baobab"))

	// loadPegPorConfigUrl is out of scope (#2576) and must keep the raw chain.
	assert.Equal(t, "https://config.orakl.network/aggregator/cypress/peg.por.json", loadPegPorConfigUrl("cypress"))
}
