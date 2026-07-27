package helper

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
)

// These tests exercise the reconcile/rotate crash-safety scenarios (issue #2516) deterministically
// with in-memory fakes for the on-chain oracle and the key store, so every failure interleaving
// (mined-but-RPC-errored, not-landed, DB-promote-failed, out-of-band removal, ...) is covered
// without a live chain or Postgres.

// ---- fakes -----------------------------------------------------------------

type fakeChain struct {
	mu           sync.Mutex
	exp          map[common.Address]time.Time
	readErr      error
	updateOracle func(f *fakeChain, from string, newAddr common.Address) error
	updateCalls  int
}

func newFakeChain() *fakeChain { return &fakeChain{exp: map[common.Address]time.Time{}} }

func (f *fakeChain) ReadExpiration(_ context.Context, addr common.Address) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readErr != nil {
		return time.Time{}, f.readErr
	}
	if v, ok := f.exp[addr]; ok {
		return v, nil
	}
	return time.Unix(0, 0), nil // never whitelisted == on-chain expirationTime 0 (matches the real contract)
}

func (f *fakeChain) UpdateOracle(_ context.Context, from string, newAddr common.Address) error {
	f.mu.Lock()
	f.updateCalls++
	fn := f.updateOracle
	f.mu.Unlock()
	if fn != nil {
		return fn(f, from, newAddr)
	}
	f.setWhitelisted(newAddr) // default: mines successfully
	return nil
}

func (f *fakeChain) setWhitelisted(addr common.Address) {
	f.mu.Lock()
	f.exp[addr] = time.Now().Add(100 * 24 * time.Hour)
	f.mu.Unlock()
}
func (f *fakeChain) setExp(addr common.Address, d time.Duration) {
	f.mu.Lock()
	f.exp[addr] = time.Now().Add(d)
	f.mu.Unlock()
}
func (f *fakeChain) setDeactivated(addr common.Address) {
	f.mu.Lock()
	f.exp[addr] = time.Now().Add(-time.Hour)
	f.mu.Unlock()
}

type fakeStore struct {
	mu         sync.Mutex
	keys       []utils.SignerKey
	legacy     string
	nextID     int64
	promoteErr error
}

func newFakeStore() *fakeStore { return &fakeStore{} }

func (s *fakeStore) addKey(pkHex string, addr common.Address, state string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	a := strings.ToLower(addr.Hex())
	s.keys = append(s.keys, utils.SignerKey{ID: s.nextID, Address: &a, PK: strings.TrimPrefix(pkHex, "0x"), State: state})
	return s.nextID
}

func (s *fakeStore) LoadKeys(_ context.Context) ([]utils.SignerKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]utils.SignerKey, len(s.keys))
	copy(out, s.keys)
	return out, nil
}
func (s *fakeStore) LoadLegacyPk(_ context.Context) (string, error) { return s.legacy, nil }
func (s *fakeStore) InsertPending(_ context.Context, address, pkHex string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	a := strings.ToLower(address)
	s.keys = append(s.keys, utils.SignerKey{ID: s.nextID, Address: &a, PK: strings.TrimPrefix(pkHex, "0x"), State: "pending"})
	return s.nextID, nil
}
func (s *fakeStore) BackfillAddress(_ context.Context, id int64, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.keys {
		if s.keys[i].ID == id {
			a := strings.ToLower(address)
			s.keys[i].Address = &a
		}
	}
	return nil
}
func (s *fakeStore) Promote(_ context.Context, id int64, pkHex string) error {
	if s.promoteErr != nil {
		return s.promoteErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.keys {
		if s.keys[i].State == "active" && s.keys[i].ID != id {
			s.keys[i].State = "retired"
		}
	}
	for i := range s.keys {
		if s.keys[i].ID == id {
			s.keys[i].State = "active"
		}
	}
	s.legacy = strings.TrimPrefix(pkHex, "0x")
	return nil
}
func (s *fakeStore) GCRetired(_ context.Context) error { return nil }

func (s *fakeStore) count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.keys) }
func (s *fakeStore) findState(state string) *utils.SignerKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.keys {
		if s.keys[i].State == state {
			k := s.keys[i]
			return &k
		}
	}
	return nil
}

// ---- helpers ---------------------------------------------------------------

func genKey(t *testing.T) (string, common.Address, *ecdsa.PrivateKey) {
	t.Helper()
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("genKey: %v", err)
	}
	return hex.EncodeToString(crypto.FromECDSA(pk)), crypto.PubkeyToAddress(pk.PublicKey), pk
}

func newScenarioSigner(chain oracleChain, store signerStore) *Signer {
	return &Signer{
		chain:              chain,
		store:              store,
		renewThreshold:     7 * 24 * time.Hour,
		livenessInterval:   30 * time.Second,
		skewMargin:         90 * time.Second,
		confirmationTTL:    90 * time.Second,
		verifyPollInterval: time.Millisecond,
		verifyPollMax:      3,
	}
}

func addrOf(pk *ecdsa.PrivateKey) common.Address { return crypto.PubkeyToAddress(pk.PublicKey) }

// ---- scenarios -------------------------------------------------------------

// S6 — the actual incident: DB + chain moved to newKey, but the running process still holds the
// old key. reconcile must hot-swap to the whitelisted held key with NO restart.
func TestScenario_IncidentSelfHeal(t *testing.T) {
	ctx := context.Background()
	oldHex, oldAddr, oldPK := genKey(t)
	newHex, newAddr, _ := genKey(t)

	store := newFakeStore()
	store.addKey(newHex, newAddr, "active")
	store.addKey(oldHex, oldAddr, "retired")
	store.legacy = newHex

	chain := newFakeChain()
	chain.setWhitelisted(newAddr)
	chain.setDeactivated(oldAddr)

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = oldPK, oldAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.activeAddr != newAddr || addrOf(s.PK) != newAddr {
		t.Fatalf("expected hot-swap to new key %s, got %s", newAddr.Hex(), s.activeAddr.Hex())
	}
	if !s.usable {
		t.Fatal("expected usable after adopting whitelisted key")
	}
}

// S10 — no held key whitelisted: refuse (fail loud), keep keys.
func TestScenario_NoneWhitelistedRefuses(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain() // aAddr absent => not whitelisted

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.usable {
		t.Fatal("expected usable=false (refuse) when no held key is whitelisted")
	}
	if store.count() != 1 {
		t.Fatal("must not delete keys when refusing")
	}
}

// S8 (fix) — active key positively deactivated while ANOTHER candidate's read fails: must still
// refuse immediately (a transient read error on some other key must not mask a real removal).
func TestScenario_ActiveDeactivatedOtherUnknownRefuses(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	bHex, bAddr, _ := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.addKey(bHex, bAddr, "pending")

	chain := newFakeChain()
	chain.setDeactivated(aAddr) // active positively NOT whitelisted
	// Make ONLY b's read fail by returning a global read error after seeding a is not possible with
	// one flag; instead model: a deactivated (present, past), b unknown via a per-addr miss is "not
	// whitelisted", so to get UNKNOWN we use readErr only for the classification of b. Simpler: use a
	// chain that errors for b but not a.
	chain.readErr = nil
	// emulate b unknown: a custom chain wrapper
	uc := &perAddrErrChain{fakeChain: chain, errAddr: bAddr, err: errors.New("rpc timeout")}

	s := newScenarioSigner(uc, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.usable {
		t.Fatal("expected refuse: active key positively deactivated even though another read failed")
	}
}

// perAddrErrChain returns an error only for errAddr, delegating everything else to fakeChain.
type perAddrErrChain struct {
	*fakeChain
	errAddr common.Address
	err     error
}

func (c *perAddrErrChain) ReadExpiration(ctx context.Context, addr common.Address) (time.Time, error) {
	if addr == c.errAddr {
		return time.Time{}, c.err
	}
	return c.fakeChain.ReadExpiration(ctx, addr)
}

// S8 (transient) — the active key's own read fails (RPC blip): keep state, do not flip usable
// (the sign gate's confirmationTTL bounds the stale window separately).
func TestScenario_ActiveUnknownKeepsState(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.readErr = errors.New("rpc down")

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !s.usable {
		t.Fatal("expected usable to be retained on a transient read failure")
	}
}

// S11 — multi-oracle network: node adopts ITS whitelisted key and ignores peers' oracle entries.
func TestScenario_MultiOracleAdoptsOwnKey(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	_, peerAddr, _ := genKey(t) // a peer oracle we do not hold
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.setWhitelisted(aAddr)
	chain.setWhitelisted(peerAddr) // peer also whitelisted; must be irrelevant

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(100*24*time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !s.usable || s.activeAddr != aAddr {
		t.Fatal("expected to keep own whitelisted key in a multi-oracle set")
	}
}

// Rotation happy path: active key near expiry -> rotate to a fresh key, confirmed on-chain.
func TestScenario_RotateHappyPath(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.legacy = aHex
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour) // whitelisted but within the 7d renew threshold

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.activeAddr == aAddr {
		t.Fatal("expected rotation to a new key")
	}
	if !s.usable {
		t.Fatal("expected usable after rotation")
	}
	if exp, _ := chain.ReadExpiration(ctx, s.activeAddr); !exp.After(time.Now()) {
		t.Fatal("new active key must be whitelisted on-chain")
	}
}

// S2 — updateOracle mines but the RPC returns an error: rotation must still succeed by reading
// chain state (the exact root cause of the incident).
func TestScenario_RotateMinedButAckError(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour)
	chain.updateOracle = func(f *fakeChain, _ string, newAddr common.Address) error {
		f.setWhitelisted(newAddr)              // it MINED
		return errors.New("http2 GOAWAY")      // but the ack errored
	}

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.activeAddr == aAddr || !s.usable {
		t.Fatal("expected rotation to succeed via chain-state confirmation despite ack error")
	}
}

// Not-landed: updateOracle never mines and the old key is still valid -> resume signing with the
// old key, and NEVER delete the persisted pending key (S1 key-loss guard).
func TestScenario_RotateNotLandedResumesOldAndKeepsPending(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour)
	chain.updateOracle = func(_ *fakeChain, _ string, _ common.Address) error {
		return errors.New("not mined") // does NOT set exp
	}

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	_ = s.reconcile(ctx, true)
	if s.activeAddr != aAddr || !s.usable {
		t.Fatal("expected to resume signing with the still-whitelisted old key")
	}
	if store.findState("pending") == nil {
		t.Fatal("persist-first: the new pending key must be retained, never deleted")
	}
}

// S1 — broadcast then the tx is not observed (crash / RPC blackout), then it mines late: the
// persisted pending key must be adopted on a later reconcile; the key is never lost.
func TestScenario_PersistFirstAdoptsLateMinedKey(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour)
	chain.updateOracle = func(_ *fakeChain, _ string, _ common.Address) error { return errors.New("blackout") }

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)
	_ = s.reconcile(ctx, true)

	pending := store.findState("pending")
	if pending == nil {
		t.Fatal("pending key lost")
	}
	pendingAddr := common.HexToAddress(*pending.Address)

	// The broadcast tx mines late; the old key is now deactivated on-chain.
	chain.setWhitelisted(pendingAddr)
	chain.setDeactivated(aAddr)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.activeAddr != pendingAddr || !s.usable {
		t.Fatal("expected to adopt the late-mined key that was persisted before broadcast")
	}
}

// S3/S9 — the rotation lands on-chain but the DB promote fails: the node must adopt the confirmed
// new key in memory anyway (adopt-before-promote) and keep signing.
func TestScenario_RotateAdoptsEvenIfPromoteFails(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.promoteErr = errors.New("db blip")
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour) // near expiry -> triggers rotation

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	_ = s.reconcile(ctx, true)
	if s.activeAddr == aAddr || !s.usable {
		t.Fatal("expected in-memory adoption of the confirmed new key even when DB promote fails")
	}
}

// Concurrency: many concurrent signs racing repeated reconciles must be data-race-free (run with
// -race). Exercises the mu (sign-path fields) vs rotateMu (reconcile) locking discipline.
func TestScenario_ConcurrentSignAndReconcile(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.legacy = aHex
	chain := newFakeChain()
	chain.setWhitelisted(aAddr) // far-future; no rotation

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(100*24*time.Hour)
	s.lastConfirmedAt = time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_, _ = s.MakeGlobalAggregateProof(1, time.Now(), "BTC-USDT")
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = s.reconcile(ctx, true)
			}
		}()
	}
	wg.Wait()
}

// S12 — an interrupted rotation left a never-mined pending key: the next rotation must RESUME it
// (reuse), not mint yet another key (no unbounded pending growth / wedge).
func TestScenario_RotateResumesExistingPending(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	bHex, bAddr, _ := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.addKey(bHex, bAddr, "pending") // leftover pending from an interrupted rotation
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour) // near expiry -> rotate
	// bAddr absent from chain.exp => not whitelisted, exp==0 => reusable

	before := store.count()
	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if store.count() != before {
		t.Fatalf("expected to reuse the existing pending key, but a new one was minted (%d -> %d)", before, store.count())
	}
	if s.activeAddr != bAddr || !s.usable {
		t.Fatal("expected to adopt the resumed pending key after it mined")
	}
}
