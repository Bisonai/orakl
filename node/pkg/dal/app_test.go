package dal

import (
	"testing"

	"bisonai.com/miko/node/pkg/utils/configfetch"
	"github.com/stretchr/testify/assert"
)

// TestBaseMikoConfigUrl verifies the config-bundle URL resolves to the
// chain-renamed `_feeds.json` form for the mainnet/kairos chain values.
func TestBaseMikoConfigUrl(t *testing.T) {
	t.Setenv("CONFIG_BASE_URL", "")
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", configfetch.URL(configfetch.FeedsPath("mainnet")))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", configfetch.URL(configfetch.FeedsPath("kairos")))
}
