package balance

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	addrA = "0x5aD1dc8e413c2c3364294d784aE8c9FafD43f079"
	addrB = "0x1AD018aa154cA85E98A49Ba04344212350A8754b"
)

func TestParseWallets(t *testing.T) {
	ws, err := ParseWallets(" " + addrA + ":10000:fee-payer , " + addrB + ":50.5 ,")
	require.NoError(t, err)
	require.Len(t, ws, 2)
	assert.Equal(t, addrA, ws[0].Address.Hex())
	assert.Equal(t, 10000.0, ws[0].Minimum)
	assert.Equal(t, "fee-payer", ws[0].Tag)
	assert.Equal(t, addrB, ws[1].Address.Hex())
	assert.Equal(t, 50.5, ws[1].Minimum)
	assert.Equal(t, defaultRole, ws[1].Tag)
	assert.NotNil(t, ws[1].BalanceHistory)
}

func TestParseWalletsRejects(t *testing.T) {
	for name, raw := range map[string]string{
		"empty":          "",
		"only commas":    " , ,",
		"no minimum":     addrA,
		"too many parts": addrA + ":1:reporter:x",
		"bad address":    "0x1234:1",
		"bad minimum":    addrA + ":lots",
		"negative":       addrA + ":-1",
		"bad role":       addrA + ":1:Reporter One",
		"duplicate":      addrA + ":1," + addrA + ":2",
		"duplicate case": addrA + ":1,0x5ad1dc8e413c2c3364294d784ae8c9fafd43f079:2",
	} {
		_, err := ParseWallets(raw)
		assert.Error(t, err, name)
	}
}

func TestWalletMetrics(t *testing.T) {
	ws, err := ParseWallets(addrA + ":100:reporter")
	require.NoError(t, err)
	registerWalletMetrics(ws)
	assert.Equal(t, 100.0, testutil.ToFloat64(minimumGauge.WithLabelValues(addrA, "reporter")))

	at := time.Unix(1_700_000_000, 0)
	recordBalance(ws[0], 87.5, at)
	assert.Equal(t, 87.5, testutil.ToFloat64(balanceGauge.WithLabelValues(addrA, "reporter")))
	assert.Equal(t, float64(at.Unix()), testutil.ToFloat64(lastSuccessGauge.WithLabelValues(addrA, "reporter")))
}
