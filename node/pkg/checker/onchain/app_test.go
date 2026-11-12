package onchain

import (
	"context"
	"errors"
	"math/big"
	"testing"

	kaia "github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type round struct {
	answer    int64
	updatedAt int64
}

type fakeClient struct {
	t         *testing.T
	r         *Reader
	head      uint64
	logs      []types.Log
	feeds     map[common.Hash]common.Address
	rounds    map[common.Address]round
	logsErr   error
	lastQuery kaia.FilterQuery
}

func (f *fakeClient) BlockNumber(context.Context) (*big.Int, error) {
	return new(big.Int).SetUint64(f.head), nil
}

func (f *fakeClient) FilterLogs(_ context.Context, q kaia.FilterQuery) ([]types.Log, error) {
	f.lastQuery = q
	if f.logsErr != nil {
		return nil, f.logsErr
	}
	var out []types.Log
	for _, l := range f.logs {
		if l.BlockNumber >= q.FromBlock.Uint64() && l.BlockNumber <= q.ToBlock.Uint64() {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeClient) HeaderByNumber(_ context.Context, n *big.Int) (*types.Header, error) {
	return &types.Header{Number: n, Time: new(big.Int).SetInt64(1_700_000_000 + n.Int64())}, nil
}

func (f *fakeClient) CallContract(_ context.Context, call kaia.CallMsg, _ *big.Int) ([]byte, error) {
	feedsM := f.r.feedsABI.Methods["feeds"]
	roundM := f.r.latestRoundABI.Methods["latestRoundData"]
	switch {
	case string(call.Data[:4]) == string(feedsM.ID):
		args, err := feedsM.Inputs.Unpack(call.Data[4:])
		require.NoError(f.t, err)
		return feedsM.Outputs.Pack(f.feeds[common.Hash(args[0].([32]byte))])
	case string(call.Data[:4]) == string(roundM.ID):
		rd := f.rounds[*call.To]
		return roundM.Outputs.Pack(uint64(1), big.NewInt(rd.answer), big.NewInt(rd.updatedAt))
	}
	return nil, errors.New("unexpected call")
}

func hashOf(name string) common.Hash { return crypto.Keccak256Hash([]byte(name)) }

func ms(v int) *int { return &v }

func updateLog(addr common.Address, block uint64, index uint, answer *big.Int) types.Log {
	return types.Log{
		Address:     addr,
		BlockNumber: block,
		Index:       index,
		Topics:      []common.Hash{feedUpdatedTopic, common.BigToHash(new(big.Int).And(answer, maxUint256()))},
	}
}

func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

func setup(t *testing.T) (*Reader, *fakeClient, common.Address, common.Address) {
	t.Helper()
	btc := common.HexToAddress("0x1000000000000000000000000000000000000001")
	eth := common.HexToAddress("0x1000000000000000000000000000000000000002")
	fc := &fakeClient{
		t:    t,
		head: 100,
		feeds: map[common.Hash]common.Address{
			hashOf("TEST-BTC"): btc,
			hashOf("TEST-ETH"): eth,
		},
		rounds: map[common.Address]round{
			btc: {answer: 7_000_000_000_000, updatedAt: 1_699_999_990},
			eth: {answer: 300_000_000_000, updatedAt: 1_699_999_995},
		},
	}
	r, err := NewReader(fc, common.HexToAddress("0x2000000000000000000000000000000000000000"))
	require.NoError(t, err)
	fc.r = r
	t.Cleanup(func() {
		for _, n := range []string{"TEST-BTC", "TEST-ETH", "TEST-NOPE"} {
			deleteFeedMetrics(n)
		}
	})
	return r, fc, btc, eth
}

func TestDecodeAnswerNegative(t *testing.T) {
	neg := big.NewInt(-12345)
	topic := common.BigToHash(new(big.Int).And(neg, maxUint256()))
	assert.Equal(t, int64(-12345), decodeAnswer(topic).Int64())
	assert.Equal(t, int64(42), decodeAnswer(common.BigToHash(big.NewInt(42))).Int64())
}

func TestApplyConfigsResolvesAndSeeds(t *testing.T) {
	r, _, btc, _ := setup(t)
	ctx := context.Background()

	err := r.ApplyConfigs(ctx, []Config{
		{Name: "TEST-BTC", SubmitInterval: ms(15000)},
		{Name: "TEST-ETH", SubmitInterval: ms(60000)},
		{Name: "TEST-NOPE", SubmitInterval: ms(60000)}, // not registered -> skipped
	})
	require.NoError(t, err)

	assert.Len(t, r.byAddress, 2)
	assert.Equal(t, "TEST-BTC", r.byAddress[btc].name)
	assert.Equal(t, 7e12, testutil.ToFloat64(answerGauge.WithLabelValues("TEST-BTC")))
	assert.Equal(t, 1_699_999_990.0, testutil.ToFloat64(updatedGauge.WithLabelValues("TEST-BTC")))
	assert.Equal(t, 15.0, testutil.ToFloat64(expectedIntervalGauge.WithLabelValues("TEST-BTC")))
	assert.Equal(t, 2.0, testutil.ToFloat64(feedsGauge))
}

func TestPollAppliesLogsInOrder(t *testing.T) {
	r, fc, btc, eth := setup(t)
	ctx := context.Background()
	require.NoError(t, r.ApplyConfigs(ctx, []Config{{Name: "TEST-BTC", SubmitInterval: ms(15000)}, {Name: "TEST-ETH", SubmitInterval: ms(60000)}}))

	// first poll only sets the starting block
	require.NoError(t, r.Poll(ctx))
	assert.Equal(t, uint64(100), r.lastBlock)

	fc.head = 110
	fc.logs = []types.Log{
		updateLog(btc, 108, 0, big.NewInt(7_100_000_000_000)),
		updateLog(btc, 105, 3, big.NewInt(7_050_000_000_000)),
		updateLog(eth, 107, 1, big.NewInt(-1)),
		updateLog(btc, 99, 0, big.NewInt(1)), // before range, ignored
	}
	before := testutil.ToFloat64(updatesCounter.WithLabelValues("TEST-BTC"))
	require.NoError(t, r.Poll(ctx))

	assert.Equal(t, uint64(101), fc.lastQuery.FromBlock.Uint64())
	assert.Equal(t, uint64(110), fc.lastQuery.ToBlock.Uint64())
	assert.Equal(t, 7.1e12, testutil.ToFloat64(answerGauge.WithLabelValues("TEST-BTC")), "latest block wins")
	assert.Equal(t, float64(1_700_000_000+108), testutil.ToFloat64(updatedGauge.WithLabelValues("TEST-BTC")))
	assert.Equal(t, before+2, testutil.ToFloat64(updatesCounter.WithLabelValues("TEST-BTC")))
	assert.Equal(t, -1.0, testutil.ToFloat64(answerGauge.WithLabelValues("TEST-ETH")))
	assert.Equal(t, uint64(110), r.lastBlock)
}

func TestPollErrorDoesNotAdvance(t *testing.T) {
	r, fc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, r.ApplyConfigs(ctx, []Config{{Name: "TEST-BTC", SubmitInterval: ms(15000)}}))
	require.NoError(t, r.Poll(ctx))

	fc.head = 120
	fc.logsErr = errors.New("rpc down")
	require.Error(t, r.Poll(ctx))
	assert.Equal(t, uint64(100), r.lastBlock)

	fc.logsErr = nil
	require.NoError(t, r.Poll(ctx))
	assert.Equal(t, uint64(101), fc.lastQuery.FromBlock.Uint64(), "retries the same range")
}

func TestPollCapsBlockRange(t *testing.T) {
	r, fc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, r.ApplyConfigs(ctx, []Config{{Name: "TEST-BTC", SubmitInterval: ms(15000)}}))
	require.NoError(t, r.Poll(ctx))

	fc.head = 100 + 5000
	require.NoError(t, r.Poll(ctx))
	assert.Equal(t, uint64(MaxBlockRange), fc.lastQuery.ToBlock.Uint64()-fc.lastQuery.FromBlock.Uint64()+1)
	assert.Equal(t, fc.head, r.lastBlock)
}

func TestApplyConfigsDropsRemovedFeeds(t *testing.T) {
	r, _, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, r.ApplyConfigs(ctx, []Config{{Name: "TEST-BTC", SubmitInterval: ms(15000)}, {Name: "TEST-ETH", SubmitInterval: ms(60000)}}))
	require.NoError(t, r.ApplyConfigs(ctx, []Config{{Name: "TEST-BTC", SubmitInterval: ms(15000)}}))

	assert.Len(t, r.byAddress, 1)
	assert.Equal(t, 1, testutil.CollectAndCount(answerGauge, "orakl_onchain_answer"), "only TEST-BTC left, TEST-ETH series removed")
}
