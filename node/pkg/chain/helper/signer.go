package helper

import (
	"context"
	"os"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/secrets"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

type SignerConfig struct {
	pk             string
	renewInterval  time.Duration
	renewThreshold time.Duration
}

type SignerOption func(*SignerConfig)

func WithSignerPk(pk string) SignerOption {
	return func(config *SignerConfig) {
		config.pk = pk
	}
}

func WithRenewInterval(renewInterval time.Duration) SignerOption {
	return func(config *SignerConfig) {
		config.renewInterval = renewInterval
	}
}

func WithRenewThreshold(renewThreshold time.Duration) SignerOption {
	return func(config *SignerConfig) {
		config.renewThreshold = renewThreshold
	}
}

// whitelist status of a candidate key, as observed on-chain.
type wlStatus int

const (
	wlUnknown     wlStatus = iota // on-chain read failed (transient RPC error)
	wlWhitelisted                 // expirationTime > now
	wlNot                         // expirationTime <= now (deactivated or never set)
)

// signerCandidate is a private key the node holds and could sign with. id is nil for a key
// that lives only in the legacy `signer` row and has not yet been copied into signer_key.
type signerCandidate struct {
	id     *int64
	addr   common.Address
	pkHex  string // decrypted, 0x-trimmed
	state  string // signer_key.state, or "legacy"
	exp    time.Time
	status wlStatus
}

// getSignerPk returns a bootstrap private key (0x-trimmed) from, in order: an operator-supplied
// key, the legacy `signer` row, or the Vault SIGNER_PK secret. It is used ONLY to seed an empty
// keyring; it does not decide which key signs (the on-chain whitelist does — see reconcile).
func getSignerPk(ctx context.Context, config SignerConfig) (string, error) {
	if config.pk != "" {
		return strings.TrimPrefix(config.pk, "0x"), nil
	}

	pk, err := utils.LoadSignerPk(ctx)
	if err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to load signer from pgs")
	}
	if pk != "" {
		return strings.TrimPrefix(pk, "0x"), nil
	}

	pk = secrets.GetSecret(SignerPk)
	if pk == "" {
		log.Error().Str("Player", "Signer").Msg("signer pk not set")
		return "", errorSentinel.ErrChainSignerPKNotFound
	}
	return strings.TrimPrefix(pk, "0x"), nil
}

func NewSigner(ctx context.Context, opts ...SignerOption) (*Signer, error) {
	config := SignerConfig{
		renewInterval:  DefaultSignerRenewInterval,
		renewThreshold: DefaultSignerRenewThreshold,
	}
	for _, opt := range opts {
		opt(&config)
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		log.Error().Str("Player", "Signer").Msg("SUBMISSION_PROXY_CONTRACT not found, signer initialization failed")
		return nil, errorSentinel.ErrChainSubmissionProxyContractNotFound
	}

	// Explicit static-key mode: an operator-supplied key (WithSignerPk) is used directly, with no
	// keyring, no rotation and no on-chain whitelist gate. Production (the aggregator) never sets
	// WithSignerPk and always uses the managed reconcile path below; this branch is for explicit
	// overrides and tests, and preserves their pre-fix behavior.
	if config.pk != "" {
		return newStaticSigner(config)
	}

	// Seed the keyring from a bootstrap key if it is empty (fresh node). Existing nodes already
	// have the legacy key migrated into signer_key by migration 000027.
	initialPkHex, err := ensureKeyring(ctx, config)
	if err != nil {
		return nil, err
	}

	initialPK, err := utils.StringToPk(initialPkHex)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to convert pk")
		return nil, err
	}

	chain, err := newRealOracleChain(ctx, submissionProxyContractAddr, initialPkHex)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to set up on-chain oracle client")
		return nil, err
	}

	livenessInterval := DefaultSignerLivenessInterval
	if raw := os.Getenv("SIGNER_LIVENESS_INTERVAL"); raw != "" {
		if d, perr := time.ParseDuration(raw); perr == nil {
			livenessInterval = d
		}
	}
	skewMargin := DefaultSignerSkewMargin
	if raw := os.Getenv("SIGNER_SKEW_MARGIN"); raw != "" {
		if d, perr := time.ParseDuration(raw); perr == nil {
			skewMargin = d
		}
	}

	s := &Signer{
		PK:               initialPK,
		chain:            chain,
		store:            realSignerStore{},
		activeAddr:       crypto.PubkeyToAddress(initialPK.PublicKey),
		usable:           false, // remains false until reconcile confirms a whitelisted key
		bootstrapPk:      config.pk,
		renewThreshold:   config.renewThreshold,
		livenessInterval: livenessInterval,
		skewMargin:       skewMargin,
		// If the active key cannot be positively re-confirmed whitelisted within this window
		// (e.g. sustained RPC failures while the oracle was removed out-of-band), the sign gate
		// refuses — bounding any stale-signing window instead of trusting a far-future cache.
		confirmationTTL:    3 * livenessInterval,
		verifyPollInterval: signerVerifyPollInterval,
		verifyPollMax:      signerVerifyPollMax,
	}

	// Establish the truth (which held key is on-chain-whitelisted) BEFORE we start signing or
	// ticking. Skip rotation on this initial pass (allowRotate=false) so a due renewal cannot
	// block startup for up to the ~60s on-chain verify budget; the liveness loop rotates. A
	// branch-b outcome (no whitelisted key yet) is not fatal — the node stays up, refuses to
	// sign, and self-heals on the liveness loop.
	if rErr := s.reconcile(ctx, false); rErr != nil {
		log.Error().Str("Player", "Signer").Err(rErr).Msg("initial signer reconcile failed; starting anyway, will retry")
	}

	go s.reconcileLoop(ctx)

	return s, nil
}

// ensureKeyring guarantees at least one key exists in signer_key and returns a key (0x-trimmed)
// suitable for building the initial read/write chainHelper. reconcile then picks the real active key.
func ensureKeyring(ctx context.Context, config SignerConfig) (string, error) {
	keys, err := utils.LoadSignerKeys(ctx)
	if err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to load signer keys during init")
	}
	if len(keys) > 0 {
		for _, k := range keys {
			if k.State == "active" {
				return k.PK, nil
			}
		}
		return keys[0].PK, nil
	}

	// Empty keyring: seed from a bootstrap key.
	pk, err := getSignerPk(ctx, config)
	if err != nil {
		return "", err
	}
	addrHex, err := utils.StringPkToAddressHex(pk)
	if err != nil {
		return "", err
	}
	id, err := utils.InsertPendingKey(ctx, addrHex, pk)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to seed signer keyring")
		return "", err
	}
	if err := utils.PromotePendingToActive(ctx, id, pk); err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to activate seeded signer key")
	}
	return pk, nil
}

// newStaticSigner builds a Signer that signs with a fixed operator-supplied key, bypassing the
// keyring, rotation and on-chain gate. Used only when WithSignerPk is set (explicit override /
// tests). It does no chain or DB I/O — signing uses PK directly — so chain/store stay nil and
// reconcile is a no-op (guarded by staticMode).
func newStaticSigner(config SignerConfig) (*Signer, error) {
	pkHex := strings.TrimPrefix(config.pk, "0x")
	pk, err := utils.StringToPk(pkHex)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to convert pk")
		return nil, err
	}
	return &Signer{
		PK:               pk,
		activeAddr:       crypto.PubkeyToAddress(pk.PublicKey),
		usable:           true,
		staticMode:       true,                          // no reconcile/rotation/confirmation gate
		cachedExpiration: time.Now().AddDate(100, 0, 0), // static key: effectively never expires
		lastConfirmedAt:  time.Now().AddDate(100, 0, 0),
		renewThreshold:   config.renewThreshold,
		skewMargin:       DefaultSignerSkewMargin,
	}, nil
}

// MakeGlobalAggregateProof signs a global aggregate. It is the authoritative fail-loud gate:
// it refuses (returns an error, so no proof is published) whenever the node is not certain it
// holds a currently-whitelisted key — never silently signing with a stale/deactivated key.
func (s *Signer) MakeGlobalAggregateProof(val int64, timestamp time.Time, name string) ([]byte, error) {
	s.mu.RLock()
	usable, rotating, static, exp, confirmedAt, pk := s.usable, s.rotating, s.staticMode, s.cachedExpiration, s.lastConfirmedAt, s.PK
	s.mu.RUnlock()

	now := time.Now()
	staleConfirmation := !static && s.confirmationTTL > 0 && now.Sub(confirmedAt) > s.confirmationTTL
	if !usable || rotating || pk == nil || now.Add(s.skewMargin).After(exp) || staleConfirmation {
		return nil, errorSentinel.ErrChainSignerNoWhitelistedKey
	}
	return utils.MakeValueSignature(val, timestamp.UnixMilli(), name, pk)
}

func (s *Signer) reconcileLoop(ctx context.Context) {
	ticker := time.NewTicker(s.livenessInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.reconcile(ctx, true); err != nil {
				log.Warn().Str("Player", "Signer").Err(err).Msg("liveness reconcile error")
			}
		}
	}
}

// CheckAndUpdateSignerPK is the public entry point (bus RENEW_SIGNER / admin). It simply drives
// a reconcile, which both self-heals the active key and rotates when near expiry.
func (s *Signer) CheckAndUpdateSignerPK(ctx context.Context) error {
	return s.reconcile(ctx, true)
}

func (s *Signer) reconcile(ctx context.Context, allowRotate bool) error {
	if s.staticMode {
		return nil
	}
	if !s.rotateMu.TryLock() {
		// a reconcile/rotation is already in progress; skip (avoids pile-up and cross-trigger races)
		return nil
	}
	defer s.rotateMu.Unlock()
	return s.reconcileLocked(ctx, allowRotate)
}

func (s *Signer) reconcileLocked(ctx context.Context, allowRotate bool) error {
	cands, err := s.loadCandidates(ctx)
	if err != nil {
		// Transient DB error: keep current signing state; the sign gate + cachedExpiration still protect us.
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to load signer candidates; keeping current state")
		return err
	}
	if len(cands) == 0 {
		// Should not happen after NewSigner seeds the keyring; treat as loud, non-fatal.
		s.setUnusable("no signer keys held")
		return errorSentinel.ErrChainSignerNoWhitelistedKey
	}

	// Classify every held key against the on-chain whitelist.
	anyUnknown := false
	var whitelisted []*signerCandidate
	for _, c := range cands {
		exp, rErr := s.chain.ReadExpiration(ctx, c.addr)
		if rErr != nil {
			c.status = wlUnknown
			anyUnknown = true
			continue
		}
		c.exp = exp
		if exp.After(time.Now()) {
			c.status = wlWhitelisted
			whitelisted = append(whitelisted, c)
		} else {
			c.status = wlNot
		}
	}

	if len(whitelisted) > 0 {
		chosen := pickChosen(whitelisted)
		if err := s.ensureActive(ctx, chosen); err != nil {
			log.Error().Str("Player", "Signer").Err(err).Msg("failed to adopt whitelisted key")
			return err
		}
		// Renew only FROM a currently-whitelisted key, and only when near expiry. Rotation is
		// skipped on the startup pass (allowRotate=false) so it never blocks boot.
		if allowRotate && s.renewalRequired(chosen.exp) {
			reuse := pickReusablePending(cands, chosen.addr)
			if rErr := s.rotateLocked(ctx, chosen, reuse); rErr != nil {
				log.Warn().Str("Player", "Signer").Err(rErr).Msg("rotation attempt did not complete; will retry")
			}
		}
		return nil
	}

	// No held key is whitelisted. Determine the ACTIVE key's own on-chain status specifically —
	// a positively-deactivated active key must stop signing immediately even if some OTHER
	// candidate's read failed (otherwise a single transient read error would mask a real removal).
	s.mu.RLock()
	activeAddr := s.activeAddr
	s.mu.RUnlock()
	activeStatus := wlUnknown
	for _, c := range cands {
		if c.addr == activeAddr {
			activeStatus = c.status
			break
		}
	}

	if activeStatus == wlUnknown && anyUnknown {
		// Could not positively confirm the active key (transient RPC failure). Keep current state,
		// but the sign gate's confirmationTTL bounds how long the node will sign without a fresh
		// on-chain confirmation, so a sustained outage still fails loud rather than signing forever.
		log.Warn().Str("Player", "Signer").Msg("whitelist reads inconclusive; keeping current signer state (bounded by confirmationTTL)")
		return nil
	}

	// The active key is positively not whitelisted (or every held key definitively isn't):
	// refuse to sign, keep all keys durably, and alarm loudly.
	addrs := make([]string, 0, len(cands))
	for _, c := range cands {
		addrs = append(addrs, c.addr.Hex())
	}
	s.setUnusable("no held signer key is whitelisted on-chain")
	log.Error().Str("Player", "Signer").Strs("heldKeys", addrs).Str("activeSigner", activeAddr.Hex()).
		Msg("no held signer key is whitelisted on-chain — refusing to sign until owner re-whitelists a held key")
	return nil
}

// loadCandidates returns all held keys: the signer_key keyring plus the legacy `signer` row (so
// the new binary can discover and adopt a key an old binary rotated in). Backfills NULL addresses.
func (s *Signer) loadCandidates(ctx context.Context) ([]*signerCandidate, error) {
	keys, err := s.store.LoadKeys(ctx)
	if err != nil {
		return nil, err
	}

	cands := make([]*signerCandidate, 0, len(keys)+1)
	seen := map[common.Address]bool{}
	for i := range keys {
		k := keys[i]
		var addr common.Address
		if k.Address != nil && *k.Address != "" {
			addr = common.HexToAddress(*k.Address)
		} else {
			ah, e := utils.StringPkToAddressHex(k.PK)
			if e != nil {
				log.Warn().Str("Player", "Signer").Err(e).Int64("id", k.ID).Msg("skipping undecodable keyring key")
				continue
			}
			addr = common.HexToAddress(ah)
			if bErr := s.store.BackfillAddress(ctx, k.ID, addr.Hex()); bErr != nil {
				log.Warn().Str("Player", "Signer").Err(bErr).Msg("failed to backfill signer_key address")
			}
		}
		if seen[addr] {
			continue
		}
		seen[addr] = true
		id := k.ID
		cands = append(cands, &signerCandidate{id: &id, addr: addr, pkHex: k.PK, state: k.State})
	}

	if legacyPk, lErr := s.store.LoadLegacyPk(ctx); lErr == nil && legacyPk != "" {
		if ah, e := utils.StringPkToAddressHex(legacyPk); e == nil {
			addr := common.HexToAddress(ah)
			if !seen[addr] {
				cands = append(cands, &signerCandidate{addr: addr, pkHex: strings.TrimPrefix(legacyPk, "0x"), state: "legacy"})
			}
		}
	}

	return cands, nil
}

// pickChosen prefers an already-active row (avoids churn), otherwise any whitelisted candidate.
func pickChosen(whitelisted []*signerCandidate) *signerCandidate {
	for _, c := range whitelisted {
		if c.state == "active" {
			return c
		}
	}
	return whitelisted[0]
}

// pickReusablePending finds a pending key that has never been an on-chain oracle (expiration == 0)
// so an interrupted rotation is resumed instead of accumulating pending rows / minting a new key.
func pickReusablePending(cands []*signerCandidate, exclude common.Address) *signerCandidate {
	for _, c := range cands {
		if c.state == "pending" && c.status == wlNot && c.exp.Unix() == 0 && c.addr != exclude && c.id != nil {
			return c
		}
	}
	return nil
}

// ensureActive makes the chosen (whitelisted) candidate the durable active key and adopts it in
// memory. Idempotent: if it is already the active in-memory key it only refreshes the cached
// expiration.
func (s *Signer) ensureActive(ctx context.Context, chosen *signerCandidate) error {
	s.mu.RLock()
	alreadyActive := s.usable && !s.rotating && s.activeAddr == chosen.addr
	s.mu.RUnlock()
	if alreadyActive {
		s.mu.Lock()
		s.cachedExpiration = chosen.exp
		s.lastConfirmedAt = time.Now() // active key freshly re-confirmed whitelisted on-chain
		s.mu.Unlock()
		return nil
	}

	// Persist the chosen key as the single active row (copying a legacy-only key into the keyring first).
	id := chosen.id
	if id == nil {
		newID, err := s.store.InsertPending(ctx, chosen.addr.Hex(), chosen.pkHex)
		if err != nil {
			return err
		}
		id = &newID
	}
	if err := s.store.Promote(ctx, *id, chosen.pkHex); err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to promote chosen key in DB")
	}

	if err := s.adopt(chosen.pkHex, chosen.addr, chosen.exp); err != nil {
		return err
	}
	log.Info().Str("Player", "Signer").Str("activeSigner", chosen.addr.Hex()).
		Msg("adopted on-chain-whitelisted signer key")
	return nil
}

// adopt swaps the in-memory signing key to the given (confirmed-whitelisted) key and marks the
// signer usable. On-chain writes go through s.chain (which signs with the key passed per call),
// so adopt does no chain I/O. Caller must hold rotateMu.
func (s *Signer) adopt(pkHex string, addr common.Address, exp time.Time) error {
	pk, err := utils.StringToPk(pkHex)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.PK = pk
	s.activeAddr = addr
	s.cachedExpiration = exp
	s.lastConfirmedAt = time.Now() // adopting a key we just confirmed whitelisted on-chain
	s.usable = true
	s.rotating = false
	s.mu.Unlock()

	return nil
}

func (s *Signer) setUnusable(reason string) {
	s.mu.Lock()
	s.usable = false
	s.rotating = false
	s.mu.Unlock()
	_ = reason
}

func (s *Signer) renewalRequired(exp time.Time) bool {
	return time.Until(exp) < s.renewThreshold
}

// rotateLocked rotates the signer key. Caller holds rotateMu and `from` is currently whitelisted.
// Ordering is crash-safe: the new key is durably persisted BEFORE the on-chain updateOracle, the
// signer refuses to sign for the in-flight window (bias to no-proof over stale-key), success is
// judged by on-chain state (not the RPC ack), and the in-memory swap happens before the DB promote.
func (s *Signer) rotateLocked(ctx context.Context, from *signerCandidate, reuse *signerCandidate) error {
	var newHex string
	var newAddr common.Address
	var id int64

	if reuse != nil {
		newHex, newAddr, id = reuse.pkHex, reuse.addr, *reuse.id
	} else {
		_, freshHex, err := utils.NewPk(ctx)
		if err != nil {
			return err
		}
		addrHex, err := utils.StringPkToAddressHex(freshHex)
		if err != nil {
			return err
		}
		newHex = strings.TrimPrefix(freshHex, "0x")
		newAddr = common.HexToAddress(addrHex)
		// PERSIST-FIRST: the new key is durable before any chain call, so it can never be lost.
		newID, err := s.store.InsertPending(ctx, newAddr.Hex(), newHex)
		if err != nil {
			return err
		}
		id = newID
	}

	// Stop signing for the in-flight window: updateOracle deactivates `from` the moment it mines,
	// so from here we must not sign with `from`.
	s.mu.Lock()
	s.rotating = true
	s.mu.Unlock()

	// updateOracle(newAddr) is submitted with `from`'s (whitelisted) key. The RPC ack is UNTRUSTED
	// (WaitMined routinely errors on txs that actually mined — the original incident).
	if err := s.chain.UpdateOracle(ctx, from.pkHex, newAddr); err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("updateOracle ack error (ignored; verifying by chain state)")
	}

	// Confirm by reading chain state, not by trusting the ack.
	var newExp time.Time
	landed := false
	for i := 0; i < s.verifyPollMax; i++ {
		exp, rErr := s.chain.ReadExpiration(ctx, newAddr)
		if rErr == nil && exp.After(time.Now()) {
			newExp = exp
			landed = true
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.verifyPollInterval):
		}
	}

	if landed {
		// Adopt the new key in memory FIRST (source of on-chain truth), then persist to DB.
		if err := s.adopt(newHex, newAddr, newExp); err != nil {
			// Clear rotating so the node isn't stuck refusing forever; the next liveness reconcile
			// re-adopts the now-whitelisted new key (adopt is the only place rotating is cleared).
			s.mu.Lock()
			s.rotating = false
			s.mu.Unlock()
			return err
		}
		if err := s.store.Promote(ctx, id, newHex); err != nil {
			log.Warn().Str("Player", "Signer").Err(err).Msg("failed to promote rotated key in DB; will reconcile")
		}
		_ = s.store.GCRetired(ctx)
		log.Info().Str("Player", "Signer").Str("newSigner", newAddr.Hex()).Msg("signer key rotated")
		return nil
	}

	// Not observed on-chain within budget: NEVER delete the pending key (it may still mine). Resume
	// signing with the old key if it is still whitelisted, else refuse until the next reconcile.
	fromExp, rErr := s.chain.ReadExpiration(ctx, from.addr)
	s.mu.Lock()
	s.rotating = false
	if rErr == nil && fromExp.After(time.Now()) {
		s.usable = true
		s.cachedExpiration = fromExp
	} else {
		s.usable = false
	}
	s.mu.Unlock()
	return errorSentinel.ErrChainTransactionFail
}

// SignerStatus is the in-memory truth reported by the admin endpoint (not a DB read), so the
// endpoint can never disagree with the key that is actually signing (the 2026-07-25 symptom).
type SignerStatus struct {
	ActiveSigner string `json:"signer"`
	Usable       bool   `json:"usable"`
	Rotating     bool   `json:"rotating"`
	ExpiresAt    int64  `json:"expiresAt"`
}

func (s *Signer) Status() SignerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SignerStatus{
		ActiveSigner: s.activeAddr.Hex(),
		Usable:       s.usable,
		Rotating:     s.rotating,
		ExpiresAt:    s.cachedExpiration.Unix(),
	}
}
