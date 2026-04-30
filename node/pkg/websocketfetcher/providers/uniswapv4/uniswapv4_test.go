//nolint:all
package uniswapv4

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/chain/websocketchainreader"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- ParsePoolID ----------

func TestParsePoolID_AcceptsCanonical0xPrefix(t *testing.T) {
	in := "0x4643e3152a6ea3957e087cefbba74a1c628aa643e6b8a0edfae23a5796dde012"
	got, err := ParsePoolID(in)
	require.NoError(t, err)
	assert.Equal(t, byte(0x46), got[0])
	assert.Equal(t, byte(0x12), got[31])
}

func TestParsePoolID_AcceptsBarePrefix(t *testing.T) {
	in := "4643e3152a6ea3957e087cefbba74a1c628aa643e6b8a0edfae23a5796dde012"
	got, err := ParsePoolID(in)
	require.NoError(t, err)
	assert.Equal(t, byte(0x46), got[0])
}

func TestParsePoolID_AcceptsMixedCase(t *testing.T) {
	upper := "0x4643E3152A6EA3957E087CEFBBA74A1C628AA643E6B8A0EDFAE23A5796DDE012"
	lower := "0x4643e3152a6ea3957e087cefbba74a1c628aa643e6b8a0edfae23a5796dde012"
	upperGot, err := ParsePoolID(upper)
	require.NoError(t, err)
	lowerGot, err := ParsePoolID(lower)
	require.NoError(t, err)
	assert.Equal(t, upperGot, lowerGot, "case must not affect parse")
}

func TestParsePoolID_RejectsEmpty(t *testing.T) {
	_, err := ParsePoolID("")
	require.Error(t, err)
}

func TestParsePoolID_RejectsWrongLength(t *testing.T) {
	for _, bad := range []string{
		"0x1234",
		"0x" + strings.Repeat("ab", 16),               // 16 bytes
		"0x" + strings.Repeat("ab", 33),               // 33 bytes
		strings.Repeat("00", 31) + "00ff",             // 32 bytes encoded as 66 chars (no 0x) — too long when bare
	} {
		_, err := ParsePoolID(bad)
		assert.Error(t, err, "expected error for %q", bad)
	}
}

func TestParsePoolID_RejectsNonHex(t *testing.T) {
	bad := "0xZZ" + strings.Repeat("00", 31)
	_, err := ParsePoolID(bad)
	require.Error(t, err)
}

func TestParsePoolID_RoundTripsToHash(t *testing.T) {
	in := "0x4643e3152a6ea3957e087cefbba74a1c628aa643e6b8a0edfae23a5796dde012"
	parsed, err := ParsePoolID(in)
	require.NoError(t, err)
	asHash := kaiacommon.BytesToHash(parsed[:])
	assert.Equal(t, strings.ToLower(in), strings.ToLower(asHash.Hex()),
		"parsed bytes must round-trip back to the original hex")
}

// ---------- GetTokenPrice ----------

// TestGetTokenPrice_JPYCPoolMatchesOnchain anchors the math against a real
// on-chain snapshot.  The sqrtPriceX96 below was read live from
// StateView.getSlot0() for the JPYC/USDC 0.05% V4 pool on Polygon during
// PR review (token0 = USDC 6 decimals, token1 = JPYC 18 decimals).
//
// 1 USD ≈ 150 JPY → raw price (token1/token0) should land near 150 JPYC
// per USDC.  This test would catch regressions in the
// sqrtPriceX96-to-human-price conversion.
func TestGetTokenPrice_JPYCPoolMatchesOnchain(t *testing.T) {
	// 0x00c1369007999520f401ab9813a21fb1
	sqrtPrice, ok := new(big.Int).SetString("c1369007999520f401ab9813a21fb1", 16)
	require.True(t, ok)

	def := &common.DexFeedDefinition{
		Token0Decimals: 6,
		Token1Decimals: 18,
	}
	got, err := GetTokenPrice(sqrtPrice, def)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Plausibility band: 100..200 JPYC per USDC.  Anything outside this
	// would mean the math drifted by orders of magnitude.
	assert.Greater(t, *got, 100.0, "raw price should be ~150 JPYC per USDC")
	assert.Less(t, *got, 200.0, "raw price should be ~150 JPYC per USDC")
}

// TestGetTokenPrice_ReciprocalInverts confirms that the reciprocal flag
// inverts the price exactly — important because synthetic configs like
// JPYC-USDT depend on this being a true 1/x.
func TestGetTokenPrice_ReciprocalInverts(t *testing.T) {
	sqrtPrice, ok := new(big.Int).SetString("c1369007999520f401ab9813a21fb1", 16)
	require.True(t, ok)

	base := &common.DexFeedDefinition{Token0Decimals: 6, Token1Decimals: 18}
	reciprocal := *base
	yes := true
	reciprocal.Reciprocal = &yes

	direct, err := GetTokenPrice(sqrtPrice, base)
	require.NoError(t, err)
	inverted, err := GetTokenPrice(sqrtPrice, &reciprocal)
	require.NoError(t, err)

	// direct * inverted ≈ 1.0 (within float64 rounding).
	product := *direct * *inverted
	assert.InDelta(t, 1.0, product, 1e-9, "reciprocal must invert the direct price")
}

// TestGetTokenPrice_RejectsNil ensures we fail loudly on a nil price
// rather than panicking inside big.Float math.
func TestGetTokenPrice_RejectsNil(t *testing.T) {
	got, err := GetTokenPrice(nil, &common.DexFeedDefinition{})
	require.Error(t, err)
	assert.Nil(t, got)
}

// TestGetTokenPrice_ReciprocalOfZeroErrors avoids a silent infinity.
func TestGetTokenPrice_ReciprocalOfZeroErrors(t *testing.T) {
	yes := true
	def := &common.DexFeedDefinition{Token0Decimals: 6, Token1Decimals: 18, Reciprocal: &yes}
	_, err := GetTokenPrice(big.NewInt(0), def)
	require.Error(t, err, "reciprocal of zero must error, not produce infinity")
}

// TestGetTokenPrice_KnownSqrtPriceX96 is a deterministic sanity check:
// when sqrtPriceX96 = 2^96 (i.e. price1/price0 = 1) and decimals match,
// we must get exactly 1.0.
func TestGetTokenPrice_UnitySqrtPrice(t *testing.T) {
	one := new(big.Int).Lsh(big.NewInt(1), 96) // 2^96
	def := &common.DexFeedDefinition{Token0Decimals: 18, Token1Decimals: 18}
	got, err := GetTokenPrice(one, def)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, *got, 1e-12)
}

// TestGetTokenPrice_DecimalsApplied verifies the 10^(d1-d0) divisor.
// At sqrtPriceX96 = 2^96 the raw token1/token0 ratio is exactly 1, so the
// human price = 1 / 10^(d1-d0).  These two cases (positive and negative
// exponent) cover both directions.
func TestGetTokenPrice_DecimalsApplied(t *testing.T) {
	one := new(big.Int).Lsh(big.NewInt(1), 96)

	// d1 > d0: human = 1 / 10^(d1-d0)
	got, err := GetTokenPrice(one, &common.DexFeedDefinition{Token0Decimals: 6, Token1Decimals: 18})
	require.NoError(t, err)
	assert.InDelta(t, math.Pow(10, -12), *got, 1e-18)

	// d0 > d1: human = 10^(d0-d1)
	got, err = GetTokenPrice(one, &common.DexFeedDefinition{Token0Decimals: 18, Token1Decimals: 6})
	require.NoError(t, err)
	assert.InDelta(t, math.Pow(10, 12), *got, 1e2)
}

// ---------- extractSqrtPriceX96 ----------

func TestExtractSqrtPriceX96_Valid(t *testing.T) {
	bi := big.NewInt(123)
	tuple := []interface{}{bi, big.NewInt(0), big.NewInt(0), big.NewInt(0)}
	got, err := extractSqrtPriceX96(tuple)
	require.NoError(t, err)
	assert.Same(t, bi, got)
}

func TestExtractSqrtPriceX96_RejectsWrongShape(t *testing.T) {
	for _, bad := range []interface{}{
		nil,
		"not a tuple",
		[]interface{}{},        // empty
		[]string{"a", "b"},     // wrong slice element type
	} {
		_, err := extractSqrtPriceX96(bad)
		assert.Error(t, err, "expected error for %v", bad)
	}
}

func TestExtractSqrtPriceX96_RejectsNonBigInt(t *testing.T) {
	tuple := []interface{}{"not a *big.Int", big.NewInt(0)}
	_, err := extractSqrtPriceX96(tuple)
	require.Error(t, err)
}

// ---------- SwapEventTopic0 ----------

func TestSwapEventTopic0_Stable(t *testing.T) {
	// keccak256("Swap(bytes32,address,int128,int128,uint160,uint128,int24,uint24)")
	// was computed offline (and cross-checked against
	// abi.Event.ID for the same canonical signature) to lock the
	// expected value.  If anyone tweaks SWAP_EVENT or
	// SwapEventCanonicalSig in a way that changes the topic[0], this
	// test will fail loudly so the change is intentional.
	got := SwapEventTopic0()

	const expected = "0x40e9cecb9f5f1f1c5b9c97dec2917b7ee92e57ba5563708daca94dd84ad7112f"
	assert.Equal(t, expected, got.Hex(),
		"V4 Swap topic[0] hash drifted; verify SwapEventCanonicalSig against PoolManager source")
}

func TestSwapEventTopic0_Deterministic(t *testing.T) {
	// Same input, same output, every call.  Rules out hidden mutable
	// state in the helper.
	a := SwapEventTopic0()
	b := SwapEventTopic0()
	assert.Equal(t, a, b)
}

// ---------- LookupChainConfig ----------

func TestLookupChainConfig_KnownChainsHaveAllFields(t *testing.T) {
	// Every chain we've registered must expose both addresses;
	// missing one would mean the pool would dial a zero address.
	for _, ct := range []websocketchainreader.BlockchainType{
		websocketchainreader.Ethereum,
		websocketchainreader.Polygon,
		websocketchainreader.Arbitrum,
		websocketchainreader.Base,
	} {
		cfg, ok := LookupChainConfig(ct)
		require.Truef(t, ok, "chain %v missing from V4 chain configs", ct)
		assert.NotEmptyf(t, cfg.StateView, "chain %v has empty StateView", ct)
		assert.NotEmptyf(t, cfg.PoolManager, "chain %v has empty PoolManager", ct)
		assert.Truef(t, strings.HasPrefix(strings.ToLower(cfg.StateView), "0x"),
			"chain %v StateView must be 0x-prefixed", ct)
		assert.Truef(t, strings.HasPrefix(strings.ToLower(cfg.PoolManager), "0x"),
			"chain %v PoolManager must be 0x-prefixed", ct)
		assert.Lenf(t, cfg.StateView, 42, "chain %v StateView address has wrong length", ct)
		assert.Lenf(t, cfg.PoolManager, 42, "chain %v PoolManager address has wrong length", ct)
	}
}

func TestLookupChainConfig_UnknownChain(t *testing.T) {
	// Kaia has no V4 deployment — make sure unknown chains report not-found
	// rather than returning a zero-valued struct that would silently
	// dial 0x000... contracts.
	cfg, ok := LookupChainConfig(websocketchainreader.Kaia)
	assert.False(t, ok)
	assert.Equal(t, ChainConfig{}, cfg)

	cfg, ok = LookupChainConfig(websocketchainreader.BlockchainType(999))
	assert.False(t, ok)
	assert.Equal(t, ChainConfig{}, cfg)
}

// ---------- New ----------

// TestNew_PropagatesOptions verifies that DexFetcherOption configures the
// underlying common.DexFetcher as expected — pulled from common.New
// behavior, this guards against future refactors silently dropping
// fields the V4 fetcher relies on (Feeds, FeedDataBuffer, ChainReader).
func TestNew_PropagatesOptions(t *testing.T) {
	feeds := []common.Feed{{ID: 1, Name: "JPYC-USDT"}}
	buf := make(chan *common.FeedData, 4)
	cr := &websocketchainreader.ChainReader{}

	f := New(
		common.WithFeeds(feeds),
		common.WithDexFeedDataBuffer(buf),
		common.WithWebsocketChainReader(cr),
	)
	v4, ok := f.(*V4Fetcher)
	require.True(t, ok)
	assert.Equal(t, feeds, v4.Feeds)
	assert.Equal(t, buf, v4.FeedDataBuffer)
	assert.Same(t, cr, v4.WebsocketChainReader)
	assert.NotNil(t, v4.LatestEntries)
	assert.Empty(t, v4.LatestEntries)
}
