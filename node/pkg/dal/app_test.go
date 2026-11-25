package dal

import (
	"testing"

	"bisonai.com/orakl/node/pkg/utils/configfetch"
	"github.com/stretchr/testify/assert"
)

// TestBaseOraklConfigUrl verifies the config-bundle URL resolves to the
// chain-renamed `_feeds.json` form for the mainnet/kairos chain values.
func TestBaseOraklConfigUrl(t *testing.T) {
	t.Setenv("CONFIG_BASE_URL", "")
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", configfetch.URL(configfetch.FeedsPath("mainnet")))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", configfetch.URL(configfetch.FeedsPath("kairos")))
}
