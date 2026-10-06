package dal

import (
	"fmt"
	"testing"

	chainname "bisonai.com/miko/node/pkg/chain"
	"github.com/stretchr/testify/assert"
)

// TestBaseMikoConfigUrl verifies the migrated config-bundle URL resolves to the
// new chain-renamed `_feeds.json` form both before and after infra flips CHAIN
// from the old value (cypress/baobab) to the new one (mainnet/kairos).
func TestBaseMikoConfigUrl(t *testing.T) {
	build := func(chain string) string {
		return fmt.Sprintf(baseMikoConfigUrl, chainname.Normalize(chain))
	}
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", build("cypress"))
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", build("mainnet"))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", build("baobab"))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", build("kairos"))
}
