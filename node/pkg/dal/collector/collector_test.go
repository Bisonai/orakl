//nolint:all
package collector

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A flood of proof rejections within one cooldown window must collapse to a single on-demand
// whitelist refresh — this is the guard against the 2026-07-25 signer-freeze RPC storm, where every
// feed's proof was rejected ~2.5x/s and each rejection previously spawned its own getAllOracles call.
func TestClaimRefreshSlot_RateLimitsBurst(t *testing.T) {
	c := &Collector{}

	allowed := 0
	for i := 0; i < 10000; i++ {
		if c.claimRefreshSlot() {
			allowed++
		}
	}

	// All iterations run well within OnDemandRefreshCooldown, so only the first may fire.
	assert.Equal(t, 1, allowed, "burst of rejections within the cooldown must trigger exactly one refresh")
}

func TestClaimRefreshSlot_ConcurrentSingleFire(t *testing.T) {
	c := &Collector{}

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.claimRefreshSlot() {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), allowed.Load(), "concurrent rejections must not race past the cooldown claim")
}

// A fresh collector (zero lastOnDemandRefresh) must allow the first refresh immediately, so recovery
// from a legitimate rotation is not delayed by the cooldown.
func TestClaimRefreshSlot_FirstCallAllowed(t *testing.T) {
	c := &Collector{}
	assert.True(t, c.claimRefreshSlot(), "the first on-demand refresh must be allowed immediately")
	assert.False(t, c.claimRefreshSlot(), "an immediate second call is within the cooldown and must be denied")
}
