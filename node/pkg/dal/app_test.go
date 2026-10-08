package dal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBaseMikoConfigUrl verifies the config-bundle URL resolves to the
// chain-renamed `_feeds.json` form for the mainnet/kairos chain values.
func TestBaseMikoConfigUrl(t *testing.T) {
	build := func(chain string) string {
		return fmt.Sprintf(baseMikoConfigUrl, chain)
	}
	assert.Equal(t, "https://config.orakl.network/mainnet_feeds.json", build("mainnet"))
	assert.Equal(t, "https://config.orakl.network/kairos_feeds.json", build("kairos"))
}
