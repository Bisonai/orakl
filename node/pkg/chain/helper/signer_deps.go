package helper

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia/common"
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
	LoadKeys(ctx context.Context) ([]utils.SignerKey, error)
	LoadLegacyPk(ctx context.Context) (string, error)
	InsertPending(ctx context.Context, address, pkHex string) (int64, error)
	BackfillAddress(ctx context.Context, id int64, address string) error
	Promote(ctx context.Context, id int64, pkHex string) error
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

// realSignerStore wraps the package-level DB helpers.
type realSignerStore struct{}

func (realSignerStore) LoadKeys(ctx context.Context) ([]utils.SignerKey, error) {
	return utils.LoadSignerKeys(ctx)
}
func (realSignerStore) LoadLegacyPk(ctx context.Context) (string, error) {
	return utils.LoadSignerPk(ctx)
}
func (realSignerStore) InsertPending(ctx context.Context, address, pkHex string) (int64, error) {
	return utils.InsertPendingKey(ctx, address, pkHex)
}
func (realSignerStore) BackfillAddress(ctx context.Context, id int64, address string) error {
	return utils.BackfillKeyAddress(ctx, id, address)
}
func (realSignerStore) Promote(ctx context.Context, id int64, pkHex string) error {
	return utils.PromotePendingToActive(ctx, id, pkHex)
}
func (realSignerStore) GCRetired(ctx context.Context) error {
	return utils.GCRetiredKeys(ctx)
}
