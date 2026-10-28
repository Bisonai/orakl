package helper

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/utils/encryptor"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

// oracleChain and signerStore are the two external dependencies of the reconcile/rotate logic.
// They are injected into Signer so the crash-safety scenarios (issue #2516) can be exercised
// deterministically in unit tests with in-memory fakes, while production uses the real impls.

type oracleChain interface {
	// ReadExpiration returns the on-chain whitelist expirationTime for addr (zero time if never
	// whitelisted). Errors are treated as "unknown" by the caller — never as "not whitelisted".
	ReadExpiration(ctx context.Context, addr common.Address) (time.Time, error)
	// UpdateOracle submits updateOracle(newAddr) signed by signWithPkHex (the current oracle key).
	UpdateOracle(ctx context.Context, signWithPkHex string, newAddr common.Address) error
}

type signerStore interface {
	// LoadKeys returns the keyring with PKs decrypted to 0x-trimmed hex.
	LoadKeys(ctx context.Context) ([]utils.SignerKey, error)
	InsertPending(ctx context.Context, address, pkHex string) (int64, error)
	BackfillAddress(ctx context.Context, id int64, address string) error
	Promote(ctx context.Context, id int64) error
	GCRetired(ctx context.Context) error
}

// realOracleChain talks to the on-chain SubmissionProxy. Reads reuse one keyless-enough helper;
// writes build a fresh helper signed by the caller-supplied key (rare, so per-call is fine).
type realOracleChain struct {
	contractAddr string
	readHelper   *ChainHelper
}

func newRealOracleChain(ctx context.Context, contractAddr, readPkHex string) (*realOracleChain, error) {
	readHelper, err := NewChainHelper(ctx, WithReporterPk(readPkHex))
	if err != nil {
		return nil, err
	}
	return &realOracleChain{contractAddr: contractAddr, readHelper: readHelper}, nil
}

func (c *realOracleChain) ReadExpiration(ctx context.Context, addr common.Address) (time.Time, error) {
	readResult, err := c.readHelper.ReadContract(ctx, c.contractAddr, SignerDetailFuncSignature, addr)
	if err != nil {
		return time.Time{}, err
	}
	values, ok := readResult.([]interface{})
	if !ok || len(values) < 2 {
		return time.Time{}, errorSentinel.ErrChainFailedToParseContractResult
	}
	rawTimestamp, ok := values[1].(*big.Int)
	if !ok {
		return time.Time{}, errorSentinel.ErrChainFailedToParseContractResult
	}
	return time.Unix(rawTimestamp.Int64(), 0), nil
}

func (c *realOracleChain) UpdateOracle(ctx context.Context, signWithPkHex string, newAddr common.Address) error {
	ch, err := NewChainHelper(ctx, WithReporterPk(signWithPkHex))
	if err != nil {
		return err
	}
	defer ch.Close()
	return ch.SubmitDelegatedFallbackDirect(ctx, c.contractAddr, UpdateSignerFuncSignature, newAddr)
}

// retiredKeyRetention is how long retired keys are kept before GC.
const retiredKeyRetention = 90 * 24 * time.Hour

// sqliteSignerStore keeps the keyring in the node's SQLite state store. PKs
// are encrypted at rest (ENCRYPT_PASSWORD), as in the Postgres keyring.
type sqliteSignerStore struct {
	state *nodestate.Store
}

// LoadKeys skips rows whose PK cannot be decrypted (or decrypts to a key not
// matching the row's address) with a warning, so one bad row never hides a
// usable key. If no row decrypts (e.g. wrong
// ENCRYPT_PASSWORD) it errors instead of returning an empty keyring, which
// would make NewSigner seed SIGNER_PK over the live keyring.
func (s sqliteSignerStore) LoadKeys(ctx context.Context) ([]utils.SignerKey, error) {
	rows, err := s.state.SignerKeys(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]utils.SignerKey, 0, len(rows))
	for _, r := range rows {
		if r.PK == "" {
			continue
		}
		decrypted, decErr := encryptor.DecryptText(r.PK)
		if decErr != nil {
			log.Warn().Err(decErr).Int64("id", r.ID).Msg("failed to decrypt signer_key row; skipping")
			continue
		}
		pkHex := strings.TrimPrefix(decrypted, "0x")
		// AES-CTR has no auth tag: a wrong password "decrypts" to garbage, so
		// check that the result is a key matching the stored address
		addr, addrErr := utils.StringPkToAddressHex(pkHex)
		if addrErr != nil || (r.Address != nil && *r.Address != "" && !strings.EqualFold(addr, *r.Address)) {
			log.Warn().Int64("id", r.ID).Msg("signer_key row does not decrypt to its key; skipping")
			continue
		}
		r.PK = pkHex
		result = append(result, r)
	}
	if len(rows) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("none of %d signer keys could be decrypted (check ENCRYPT_PASSWORD)", len(rows))
	}
	return result, nil
}

func (s sqliteSignerStore) InsertPending(ctx context.Context, address, pkHex string) (int64, error) {
	encryptedPk, err := encryptor.EncryptText(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return 0, err
	}
	return s.state.InsertPendingKey(ctx, strings.ToLower(address), encryptedPk)
}

func (s sqliteSignerStore) BackfillAddress(ctx context.Context, id int64, address string) error {
	return s.state.BackfillKeyAddress(ctx, id, strings.ToLower(address))
}

func (s sqliteSignerStore) Promote(ctx context.Context, id int64) error {
	return s.state.ActivateKey(ctx, id)
}

func (s sqliteSignerStore) GCRetired(ctx context.Context) error {
	return s.state.GCRetiredKeys(ctx, time.Now().Add(-retiredKeyRetention))
}
