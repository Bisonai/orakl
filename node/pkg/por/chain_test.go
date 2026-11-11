package por

import (
	"testing"

	"bisonai.com/miko/node/pkg/utils/configfetch"
	"github.com/stretchr/testify/assert"
)

// TestMergedURLResolution verifies that the per-feed mag7 URL built from the
// urls map resolves to the canonical uppercase/new-chain form for the
// mainnet/kairos chain values.
func TestMergedURLResolution(t *testing.T) {
	t.Setenv("CONFIG_BASE_URL", "")
	build := func(chain, name string) string {
		return configfetch.URL(mag7Path(chain, name))
	}

	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/AAPL.json", build("mainnet", "aapl"))
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/TSLA.json", build("mainnet", "tsla"))
	assert.Equal(t, "https://config.orakl.network/mag7/kairos/AAPL.json", build("kairos", "aapl"))
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/NVDA.json", build("mainnet", "nvda"))
}
