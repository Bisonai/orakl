package helper

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"bisonai.com/miko/node/pkg/metrics/submission"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signedFeeDelegatedTx(t *testing.T) *types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    uint64(1),
		types.TxValueKeyGasPrice: big.NewInt(25),
		types.TxValueKeyGasLimit: uint64(90000),
		types.TxValueKeyTo:       common.HexToAddress("0x1234"),
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     crypto.PubkeyToAddress(key.PublicKey),
		types.TxValueKeyData:     []byte{0xf0, 0x71, 0x54, 0xbe},
		types.TxValueKeyFeePayer: common.HexToAddress("0x2"),
	})
	require.NoError(t, err)
	require.NoError(t, tx.SignWithKeys(types.NewEIP155Signer(big.NewInt(1001)), []*ecdsa.PrivateKey{key}))
	return tx
}

func TestGetSignedFromDelegatorSendsToken(t *testing.T) {
	tx := signedFeeDelegatedTx(t)
	var gotAuth []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		assert.Equal(t, DelegatorEndpoint, r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		raw := utils.TxToHash(tx)
		_ = json.NewEncoder(w).Encode(map[string]string{"signedRawTx": raw})
	}))
	defer server.Close()

	withToken := &ChainHelper{delegatorUrl: server.URL, delegatorToken: "dlg_secret"}
	_, err := withToken.GetSignedFromDelegator(tx)
	require.NoError(t, err)

	withoutToken := &ChainHelper{delegatorUrl: server.URL}
	_, err = withoutToken.GetSignedFromDelegator(tx)
	require.NoError(t, err)

	assert.Equal(t, []string{"Bearer dlg_secret", ""}, gotAuth)
}

func TestParseDelegatorToken(t *testing.T) {
	for _, tc := range []struct {
		name, raw, want string
		wantErr         bool
	}{
		{name: "empty", raw: "", want: ""},
		{name: "plain", raw: "dlg_secret", want: "dlg_secret"},
		{name: "trailing newline", raw: "dlg_secret\n", want: "dlg_secret"},
		{name: "crlf", raw: "dlg_secret\r\n", want: "dlg_secret"},
		{name: "surrounding spaces", raw: "  dlg_secret \t", want: "dlg_secret"},
		{name: "only whitespace", raw: " \n", want: ""},
		{name: "inner newline", raw: "dlg\nsecret", wantErr: true},
		{name: "inner nul", raw: "dlg\x00secret", wantErr: true},
		{name: "inner del", raw: "dlg\x7fsecret", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDelegatorToken(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), DelegatorToken)
				assert.NotContains(t, err.Error(), "secret")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// counterValue reads one series of a node counter from the default registry
// (0 when not yet registered or not present).
func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	metric:
		for _, m := range family.GetMetric() {
			for _, lp := range m.GetLabel() {
				if labels[lp.GetName()] != lp.GetValue() {
					continue metric
				}
			}
			return m.GetCounter().GetValue()
		}
	}
	return 0
}

func TestSignOrFallback(t *testing.T) {
	tx := signedFeeDelegatedTx(t)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"signedRawTx": utils.TxToHash(tx)})
	})
	// drops the connection without a response: a transport error
	hangup := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err) // not FailNow: off the test goroutine
			return
		}
		_ = conn.Close()
	})
	reasons := []string{submission.ReasonRequest, submission.ReasonHTTPStatus, submission.ReasonInvalidResponse}
	errorTotals := func() map[string]float64 {
		out := map[string]float64{}
		for _, r := range reasons {
			out[r] = counterValue(t, metricnames.ReporterDelegatorErrors, map[string]string{metricnames.ReasonLabel: r, metricnames.PurposeLabel: submission.PurposeRotate})
		}
		return out
	}

	for _, tc := range []struct {
		name     string
		handler  http.Handler // nil: no delegator url
		wantPath string
		reason   string // expected delegator error reason, "" for none
	}{
		{name: "delegated", handler: ok, wantPath: submission.PathDelegated},
		{name: "no delegator", wantPath: submission.PathDirect},
		{name: "transport error", handler: hangup, wantPath: submission.PathDirectFallback, reason: submission.ReasonRequest},
		{name: "non-200", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}), wantPath: submission.PathDirectFallback, reason: submission.ReasonHTTPStatus},
		{name: "bad json", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}), wantPath: submission.PathDirectFallback, reason: submission.ReasonInvalidResponse},
		{name: "empty signed tx", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{}"))
		}), wantPath: submission.PathDirectFallback, reason: submission.ReasonInvalidResponse},
		{name: "undecodable signed tx", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"signedRawTx":"0xzz"}`))
		}), wantPath: submission.PathDirectFallback, reason: submission.ReasonInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := &ChainHelper{}
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				ch.delegatorUrl = server.URL
			}
			before := errorTotals()

			signed, path := ch.signOrFallback(tx, submission.PurposeRotate)

			assert.Equal(t, tc.wantPath, path)
			if tc.wantPath == submission.PathDelegated {
				require.NotNil(t, signed)
			} else {
				assert.Nil(t, signed)
			}
			want := before
			if tc.reason != "" {
				want[tc.reason]++
			}
			assert.Equal(t, want, errorTotals())
		})
	}
}

func TestRateLimitedWarner(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	now := time.Unix(0, 0)
	w := newRateLimitedWarner(time.Minute, func() time.Time { return now })
	lines := func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &m))
			out = append(out, m)
		}
		return out
	}
	boom := errors.New("boom")

	w.warn(submission.ReasonHTTPStatus, submission.PurposeSubmit, boom)
	w.warn(submission.ReasonHTTPStatus, submission.PurposeSubmit, boom)
	w.warn(submission.ReasonHTTPStatus, submission.PurposeSubmit, boom)
	// another reason has its own budget
	w.warn(submission.ReasonRequest, submission.PurposeSubmit, boom)
	require.Len(t, lines(), 2)

	now = now.Add(59 * time.Second)
	w.warn(submission.ReasonHTTPStatus, submission.PurposeSubmit, boom)
	require.Len(t, lines(), 2)

	now = now.Add(time.Second)
	w.warn(submission.ReasonHTTPStatus, submission.PurposeSubmit, boom)
	got := lines()
	require.Len(t, got, 3)
	assert.Equal(t, "warn", got[0]["level"])
	assert.Equal(t, "boom", got[0]["error"])
	assert.Equal(t, submission.ReasonHTTPStatus, got[0]["reason"])
	assert.Equal(t, float64(0), got[0]["suppressed"])
	assert.Equal(t, float64(3), got[2]["suppressed"])
}
