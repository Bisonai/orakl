package por

import (
	"strings"
	"testing"

	chainname "bisonai.com/miko/node/pkg/chain"
	"github.com/stretchr/testify/assert"
)

// TestMergedURLResolution verifies that the per-feed mag7 URL built from the
// urls map resolves to the new canonical uppercase/new-chain form even while
// infra still injects the old chain value (cypress/baobab).
func TestMergedURLResolution(t *testing.T) {
	build := func(chain, name string) string {
		return mag7BaseUrl + strings.ReplaceAll(urls[name].endpoint, "{CHAIN}", chainname.Normalize(chain))
	}

	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/AAPL.json", build("cypress", "aapl"))
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/TSLA.json", build("cypress", "tsla"))
	assert.Equal(t, "https://config.orakl.network/mag7/kairos/AAPL.json", build("baobab", "aapl"))
	// already-normalized value passes through
	assert.Equal(t, "https://config.orakl.network/mag7/mainnet/NVDA.json", build("mainnet", "nvda"))
}
