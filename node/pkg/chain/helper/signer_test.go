package helper

import (
	"errors"
	"testing"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
)

// The sign gate is the linchpin safety property (issue #2516): the node must refuse to sign
// (branch b) whenever it is not certain it holds a currently-whitelisted key, and must never
// silently sign with a stale/deactivated key.

func newGateSigner(t *testing.T) *Signer {
	t.Helper()
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return &Signer{PK: pk, skewMargin: DefaultSignerSkewMargin}
}

func TestSignGateRefusals(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name             string
		usable           bool
		rotating         bool
		cachedExpiration time.Time
		nilPK            bool
	}{
		{"not usable", false, false, now.Add(time.Hour), false},
		{"rotating", true, true, now.Add(time.Hour), false},
		{"expired", true, false, now.Add(-time.Second), false},
		{"within skew margin", true, false, now.Add(10 * time.Second), false},
		{"nil pk", true, false, now.Add(time.Hour), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newGateSigner(t)
			s.usable = tc.usable
			s.rotating = tc.rotating
			s.cachedExpiration = tc.cachedExpiration
			if tc.nilPK {
				s.PK = nil
			}
			proof, err := s.MakeGlobalAggregateProof(1, now, "BTC-USDT")
			if err == nil {
				t.Fatalf("expected refusal, got proof %x", proof)
			}
			if !errors.Is(err, errorSentinel.ErrChainSignerNoWhitelistedKey) {
				t.Fatalf("expected ErrChainSignerNoWhitelistedKey, got %v", err)
			}
			if proof != nil {
				t.Fatalf("expected no proof on refusal, got %x", proof)
			}
		})
	}
}

func TestSignGateSignsWhenUsableAndValid(t *testing.T) {
	s := newGateSigner(t)
	s.usable = true
	s.cachedExpiration = time.Now().Add(24 * time.Hour)

	proof, err := s.MakeGlobalAggregateProof(200000000, time.Now(), "test-aggregate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proof) != 65 {
		t.Fatalf("expected a 65-byte ECDSA proof, got %d bytes", len(proof))
	}
}

func TestPickChosenPrefersActive(t *testing.T) {
	pending := &signerCandidate{addr: common.HexToAddress("0x1"), state: "pending"}
	active := &signerCandidate{addr: common.HexToAddress("0x2"), state: "active"}
	if got := pickChosen([]*signerCandidate{pending, active}); got != active {
		t.Fatal("expected the active-state candidate to be preferred")
	}
	if got := pickChosen([]*signerCandidate{pending}); got != pending {
		t.Fatal("expected the only candidate to be chosen when none is active")
	}
}

func TestPickReusablePending(t *testing.T) {
	id := int64(5)
	exclude := common.HexToAddress("0x00000000000000000000000000000000000000aa")

	// Reusable: pending, not whitelisted, never-an-oracle (exp unix 0), different addr, has id.
	reusable := &signerCandidate{id: &id, addr: common.HexToAddress("0x00000000000000000000000000000000000000bb"), state: "pending", status: wlNot, exp: time.Unix(0, 0)}
	// Not reusable: expired but was previously an oracle (nonzero exp) -> updateOracle would revert.
	wasOracle := &signerCandidate{id: &id, addr: common.HexToAddress("0x00000000000000000000000000000000000000cc"), state: "pending", status: wlNot, exp: time.Now().Add(-time.Hour)}
	// Not reusable: it is the active address we are rotating from.
	isActive := &signerCandidate{id: &id, addr: exclude, state: "pending", status: wlNot, exp: time.Unix(0, 0)}
	// Not reusable: no id (legacy-only) cannot be promoted by id.
	noID := &signerCandidate{addr: common.HexToAddress("0x00000000000000000000000000000000000000dd"), state: "pending", status: wlNot, exp: time.Unix(0, 0)}

	if got := pickReusablePending([]*signerCandidate{wasOracle, reusable}, exclude); got != reusable {
		t.Fatal("expected the never-mined pending key to be reusable")
	}
	if got := pickReusablePending([]*signerCandidate{wasOracle, isActive, noID}, exclude); got != nil {
		t.Fatal("expected no reusable pending among ineligible candidates")
	}
}

func TestRenewalRequired(t *testing.T) {
	s := &Signer{renewThreshold: 7 * 24 * time.Hour}
	if !s.renewalRequired(time.Now().Add(time.Hour)) {
		t.Fatal("expected renewal required when expiry is within the threshold")
	}
	if s.renewalRequired(time.Now().Add(30 * 24 * time.Hour)) {
		t.Fatal("expected no renewal when expiry is far beyond the threshold")
	}
}

func TestStatusReportsInMemoryState(t *testing.T) {
	pk, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	addr := crypto.PubkeyToAddress(pk.PublicKey)
	exp := time.Now().Add(time.Hour)
	s := &Signer{PK: pk, activeAddr: addr, usable: true, cachedExpiration: exp}

	st := s.Status()
	if st.ActiveSigner != addr.Hex() {
		t.Fatalf("expected active signer %s, got %s", addr.Hex(), st.ActiveSigner)
	}
	if !st.Usable {
		t.Fatal("expected usable=true")
	}
	if st.ExpiresAt != exp.Unix() {
		t.Fatalf("expected expiresAt %d, got %d", exp.Unix(), st.ExpiresAt)
	}
}
