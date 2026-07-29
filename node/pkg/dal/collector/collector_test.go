//nolint:all
package collector

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A fixed, realistic UnixNano so the cooldown is exercised deterministically, independent of the
// test runner's wall clock (a paused/slow runner must not cross the cooldown and flake the tests).
const fixedNow = int64(1_700_000_000_000_000_000)

// A flood of proof rejections within one cooldown window must collapse to a single on-demand
// whitelist refresh — this is the guard against the 2026-07-25 signer-freeze RPC storm, where every
// feed's proof was rejected ~2.5x/s and each rejection previously spawned its own getAllOracles call.
func TestClaimRefreshSlot_RateLimitsBurst(t *testing.T) {
	c := &Collector{}

	allowed := 0
	for i := 0; i < 10000; i++ {
		if c.claimRefreshSlotAt(fixedNow) {
			allowed++
		}
	}

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
			if c.claimRefreshSlotAt(fixedNow) {
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
	assert.True(t, c.claimRefreshSlotAt(fixedNow), "the first on-demand refresh must be allowed immediately")
	assert.False(t, c.claimRefreshSlotAt(fixedNow), "a second call at the same instant is within the cooldown and must be denied")
}

// Once the cooldown has elapsed a new refresh is allowed again, so the whitelist keeps recovering.
func TestClaimRefreshSlot_AllowsAfterCooldown(t *testing.T) {
	c := &Collector{}
	assert.True(t, c.claimRefreshSlotAt(fixedNow), "first claim allowed")
	assert.False(t, c.claimRefreshSlotAt(fixedNow+int64(OnDemandRefreshCooldown)-1), "just before the cooldown elapses, denied")
	assert.True(t, c.claimRefreshSlotAt(fixedNow+int64(OnDemandRefreshCooldown)), "once the cooldown elapses, allowed again")
}
