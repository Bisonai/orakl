package helper

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"testing"

	"bisonai.com/orakl/node/pkg/chain/noncemanagerv2"
	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFakeRPC = errors.New("fake rpc error")

// nonceFakeClient fails SuggestGasPrice on the listed call numbers (1-based)
// and fails every SendTransaction, recording the sent nonces.
type nonceFakeClient struct {
	mu          sync.Mutex
	gasCalls    int
	failGasCall map[int]bool
	sent        []uint64
}

func (c *nonceFakeClient) Close() {}
func (c *nonceFakeClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return 100, nil
}
func (c *nonceFakeClient) SuggestGasPrice(context.Context) (*big.Int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gasCalls++
	if c.failGasCall[c.gasCalls] {
		return nil, errFakeRPC
	}
	return big.NewInt(25), nil
}
func (c *nonceFakeClient) EstimateGas(context.Context, kaia.CallMsg) (uint64, error) {
	return 100000, nil
}
func (c *nonceFakeClient) SendTransaction(_ context.Context, tx *types.Transaction) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, tx.Nonce())
	return errFakeRPC
}
func (c *nonceFakeClient) CallContract(context.Context, kaia.CallMsg, *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *nonceFakeClient) NetworkID(context.Context) (*big.Int, error) { return nil, nil }
func (c *nonceFakeClient) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *nonceFakeClient) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return nil, nil
}
func (c *nonceFakeClient) BlockNumber(context.Context) (*big.Int, error) { return nil, nil }
func (c *nonceFakeClient) SubscribeFilterLogs(context.Context, kaia.FilterQuery, chan<- types.Log) (kaia.Subscription, error) {
	return nil, nil
}
func (c *nonceFakeClient) SubscribeNewHead(context.Context, chan<- *types.Header) (kaia.Subscription, error) {
	return nil, nil
}

const testSubmitFunc = "submit(uint256)"

func newNonceTestHelper(t *testing.T, failGasCall ...int) (*ChainHelper, *nonceFakeClient) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	wallet := hex.EncodeToString(crypto.FromECDSA(key))
	fail := map[int]bool{}
	for _, n := range failGasCall {
		fail[n] = true
	}
	client := &nonceFakeClient{failGasCall: fail}
	nm, err := noncemanagerv2.New(context.Background(), client, wallet)
	require.NoError(t, err)
	// no delegatorUrl: delegator signing always fails -> direct fallback
	return &ChainHelper{client: client, wallet: wallet, chainID: big.NewInt(1001), noncemanager: nm}, client
}

func TestSubmitDelegatedReleasesNonceOnDelegatedBuildError(t *testing.T) {
	h, client := newNonceTestHelper(t, 1)
	ctx := context.Background()

	err := h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	require.ErrorIs(t, err, errFakeRPC)

	// next submission must reuse nonce 100 instead of leaving a gap
	_ = h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	assert.Equal(t, []uint64{100}, client.sent)
}

func TestSubmitDelegatedReleasesNonceOnFallbackBuildError(t *testing.T) {
	// call 1 = fee-delegated build ok, call 2 = direct fallback build fails
	h, client := newNonceTestHelper(t, 2)
	ctx := context.Background()

	err := h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	require.ErrorIs(t, err, errFakeRPC)
	assert.Empty(t, client.sent)

	_ = h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	assert.Equal(t, []uint64{100}, client.sent)
}

func TestSubmitDelegatedKeepsNonceOnSendError(t *testing.T) {
	h, client := newNonceTestHelper(t)
	ctx := context.Background()

	// send errors are ambiguous: the nonce must not be reused
	_ = h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	_ = h.SubmitDelegatedFallbackDirect(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	assert.Equal(t, []uint64{100, 101}, client.sent)
}

func TestMakeDirectTxReleasesNonceOnBuildError(t *testing.T) {
	h, _ := newNonceTestHelper(t, 2)
	ctx := context.Background()

	tx, err := h.MakeDirectTx(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	require.NoError(t, err)
	assert.Equal(t, uint64(100), tx.Nonce())

	_, err = h.MakeDirectTx(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	require.ErrorIs(t, err, errFakeRPC)

	tx, err = h.MakeDirectTx(ctx, "0x1234", testSubmitFunc, big.NewInt(1))
	require.NoError(t, err)
	assert.Equal(t, uint64(101), tx.Nonce())
}
