package por

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMergedURLResolution verifies that the per-feed mag7 URL built from the
// urls map resolves to the canonical uppercase/new-chain form for the
// mainnet/kairos chain values.
func TestMergedURLResolution(t *testing.T) {
	build := func(chain, name string) string {
		return mag7BaseUrl + strings.ReplaceAll(urls[name].endpoint, "{CHAIN}", chain)
	}

	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/AAPL.json", build("mainnet", "aapl"))
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/TSLA.json", build("mainnet", "tsla"))
	assert.Equal(t, "https://config.orakl.network/mag7/kairos/AAPL.json", build("kairos", "aapl"))
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/NVDA.json", build("mainnet", "nvda"))
}
