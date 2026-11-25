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

	"bisonai.com/orakl/node/pkg/chain/utils"
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
	// notSentErr makes UpdateOracle fail before broadcast (onBroadcast is not called).
	notSentErr error
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

func (f *fakeChain) UpdateOracle(_ context.Context, from string, newAddr common.Address, onBroadcast func()) error {
	f.mu.Lock()
	f.updateCalls++
	fn, notSentErr := f.updateOracle, f.notSentErr
	f.mu.Unlock()
	if notSentErr != nil {
		return notSentErr
	}
	onBroadcast()
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
	nextID     int64
	promoteErr error
	// promoteFailN makes the next N Promote calls fail (transient DB error).
	promoteFailN int
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
	a := strings.ToLower(address)
	for i := range s.keys {
		// Mirror the real store's unique address index (signer_key_addr).
		if s.keys[i].ID != id && s.keys[i].Address != nil && *s.keys[i].Address == a {
			return errors.New("UNIQUE constraint failed: signer_key.address")
		}
	}
	for i := range s.keys {
		if s.keys[i].ID == id {
			s.keys[i].Address = &a
		}
	}
	return nil
}
func (s *fakeStore) Promote(_ context.Context, id int64) error {
	if s.promoteErr != nil {
		return s.promoteErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.promoteFailN > 0 {
		s.promoteFailN--
		return errors.New("db blip")
	}
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

		rotationBackoffBase: DefaultSignerRotationBackoffBase,
		rotationBackoffMax:  DefaultSignerRotationBackoffMax,
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

// #2856 — rotation lands but the DB promote fails once: the next reconcile (alreadyActive path)
// must retry Promote so the DB active row matches the in-memory key, without interrupting signing.
func TestScenario_RotatePromoteFailRetriedOnNextReconcile(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	store.promoteFailN = 1
	chain := newFakeChain()
	chain.setExp(aAddr, 3*24*time.Hour) // near expiry -> triggers rotation
	// updateOracle deactivates the old key and whitelists the new one (real contract behaviour).
	chain.updateOracle = func(f *fakeChain, _ string, newAddr common.Address) error {
		f.setDeactivated(aAddr)
		f.setWhitelisted(newAddr)
		return nil
	}

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)
	s.lastConfirmedAt = time.Now()

	_ = s.reconcile(ctx, true)
	newAddr := s.activeAddr
	if newAddr == aAddr || !s.usable {
		t.Fatal("expected in-memory adoption of the rotated key")
	}
	if act := store.findState("active"); act == nil || !strings.EqualFold(*act.Address, aAddr.Hex()) {
		t.Fatal("precondition: DB active row should still be the old key after the failed promote")
	}
	if _, err := s.MakeGlobalAggregateProof(1, time.Now(), "X"); err != nil {
		t.Fatalf("signing interrupted after failed promote: %v", err)
	}

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	act := store.findState("active")
	if act == nil || !strings.EqualFold(*act.Address, newAddr.Hex()) {
		t.Fatalf("expected DB active row to be adopted key %s after retry, got %+v", newAddr.Hex(), act)
	}
	if s.activeAddr != newAddr {
		t.Fatal("in-memory key changed on retry")
	}
	if _, err := s.MakeGlobalAggregateProof(1, time.Now(), "X"); err != nil {
		t.Fatalf("signing interrupted after promote retry: %v", err)
	}
}

// #2856 — ensureActive adopts a whitelisted pending key but its Promote fails once: the next
// reconcile must retry Promote on the alreadyActive path.
func TestScenario_AdoptPromoteFailRetriedOnNextReconcile(t *testing.T) {
	ctx := context.Background()
	oldHex, oldAddr, oldPK := genKey(t)
	newHex, newAddr, _ := genKey(t)
	store := newFakeStore()
	store.addKey(oldHex, oldAddr, "active")
	store.addKey(newHex, newAddr, "pending")
	store.promoteFailN = 1
	chain := newFakeChain()
	chain.setDeactivated(oldAddr)
	chain.setWhitelisted(newAddr)

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = oldPK, oldAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.activeAddr != newAddr || !s.usable {
		t.Fatal("expected adoption of whitelisted pending key")
	}
	if act := store.findState("active"); act == nil || !strings.EqualFold(*act.Address, oldAddr.Hex()) {
		t.Fatal("precondition: DB active row should still be the old key after the failed promote")
	}

	if err := s.reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	act := store.findState("active")
	if act == nil || !strings.EqualFold(*act.Address, newAddr.Hex()) {
		t.Fatalf("expected DB active row to be adopted key %s after retry, got %+v", newAddr.Hex(), act)
	}
	if _, err := s.MakeGlobalAggregateProof(1, time.Now(), "X"); err != nil {
		t.Fatalf("signing interrupted: %v", err)
	}
}

// #2856 review — duplicate rows for one address (a legacy NULL-address row, whose backfill would
// hit the real store's unique address index, plus an addressed row): the active row wins,
// so the promote retry never re-activates a stale duplicate over the active row.
func TestScenario_DuplicateAddressPrefersActiveRow(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ nullState, addrState, want string }{
		{"retired", "active", "active"},
		{"pending", "active", "active"},
		{"active", "retired", "active"},
	} {
		aHex, aAddr, _ := genKey(t)
		store := newFakeStore()
		nullID := store.addKey(aHex, aAddr, tc.nullState)
		store.keys[len(store.keys)-1].Address = nil // legacy row without address
		addrID := store.addKey(aHex, aAddr, tc.addrState)
		wantID := addrID
		if tc.nullState == tc.want {
			wantID = nullID
		}

		s := newScenarioSigner(newFakeChain(), store)
		cands, err := s.loadCandidates(ctx)
		if err != nil {
			t.Fatalf("loadCandidates: %v", err)
		}
		if len(cands) != 1 || cands[0].state != tc.want || *cands[0].id != wantID {
			t.Fatalf("%+v: expected one candidate id=%d state=%s, got %+v", tc, wantID, tc.want, cands)
		}
	}
}

// #2856 review — the alreadyActive path must not promote a stale duplicate over the active row.
func TestScenario_DuplicateAddressRetryKeepsActiveRow(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "retired")
	store.keys[len(store.keys)-1].Address = nil
	activeID := store.addKey(aHex, aAddr, "active")
	chain := newFakeChain()
	chain.setWhitelisted(aAddr)

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(time.Hour)

	if err := s.reconcile(ctx, false); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if act := store.findState("active"); act == nil || act.ID != activeID {
		t.Fatalf("expected row %d to stay active, got %+v", activeID, act)
	}
}

// Concurrency: many concurrent signs racing repeated reconciles must be data-race-free (run with
// -race). Exercises the mu (sign-path fields) vs rotateMu (reconcile) locking discipline.
func TestScenario_ConcurrentSignAndReconcile(t *testing.T) {
	ctx := context.Background()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
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
