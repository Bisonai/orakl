package sign

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"testing"

	"bisonai.com/miko/node/pkg/delegator/metrics"
	"bisonai.com/miko/node/pkg/delegator/store"
	"bisonai.com/miko/node/pkg/delegator/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metricValue reads a counter or gauge from the default registry (0 if the
// series does not exist).
func metricValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
	next:
		for _, m := range f.GetMetric() {
			got := map[string]string{}
			for _, l := range m.GetLabel() {
				got[l.GetName()] = l.GetValue()
			}
			for k, v := range labels {
				if got[k] != v {
					continue next
				}
			}
			switch {
			case m.GetCounter() != nil:
				return m.GetCounter().GetValue()
			case m.GetGauge() != nil:
				return m.GetGauge().GetValue()
			case m.GetHistogram() != nil:
				return float64(m.GetHistogram().GetSampleCount())
			}
		}
	}
	return 0
}

func signRequests(t *testing.T, result, caller string) float64 {
	return metricValue(t, "orakl_delegator_sign_requests_total", map[string]string{"result": result, "caller": caller})
}

func storeErrors(t *testing.T, op string) float64 {
	return metricValue(t, "orakl_delegator_store_errors_total", map[string]string{"op": op})
}

// assertCounted runs fn and asserts it added exactly one sign request with
// result/caller and no other result.
func assertCounted(t *testing.T, result, caller string, fn func()) {
	t.Helper()
	callers := []string{"", "node-a"}
	before := map[string]float64{}
	for _, r := range []string{metrics.ResultOK, metrics.ResultNotAllowlisted, metrics.ResultAuthFailed, metrics.ResultInvalidTx, metrics.ResultError} {
		for _, c := range callers {
			before[r+"/"+c] = signRequests(t, r, c)
		}
	}
	fn()
	for key, b := range before {
		want := b
		if key == result+"/"+caller {
			want++
		}
		r, c, _ := bytes.Cut([]byte(key), []byte("/"))
		assert.Equal(t, want, signRequests(t, string(r), string(c)), "result=%s caller=%s", r, c)
	}
}

func rawSignRequest(t *testing.T, app *fiber.App, body, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/sign/v2", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	resp.Body.Close()
	return resp.StatusCode
}

func TestSignMetricsSQLite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	contract := common.HexToAddress("0xCB56B163E545A3870CA04C6AE2401F2405FB29A9")

	s, err := store.Open(ctx, dir, 0, true)
	require.NoError(t, err)
	_, err = s.Upsert(ctx, contract.Hex(), submitSignature, "")
	require.NoError(t, err)
	token, err := s.AddToken(ctx, "node-a")
	require.NoError(t, err)
	require.NoError(t, s.Close())

	feePayerKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	feePayerPK := hex.EncodeToString(crypto.FromECDSA(feePayerKey))
	t.Setenv("DATABASE_URL", "")
	t.Setenv(store.EnvDir, dir)
	t.Setenv("DELEGATOR_FEEPAYER_PK", feePayerPK)

	appConfig, err := utils.Setup("test")
	require.NoError(t, err)
	app := appConfig.App
	SigningRoutes(app.Group("/api/v1"))

	chainID := big.NewInt(1001)
	selector := crypto.Keccak256([]byte(submitSignature))[:4]
	call := append(selector, 0x01)

	t.Run("ok", func(t *testing.T) {
		signed := metricValue(t, "orakl_delegator_sign_duration_seconds", nil)
		assertCounted(t, metrics.ResultOK, "node-a", func() {
			tx, _ := signedSenderTx(t, contract, call, chainID)
			code, body := signRequest(t, app, tx, token)
			require.Equal(t, http.StatusOK, code, body)
		})
		assert.Equal(t, signed+1, metricValue(t, "orakl_delegator_sign_duration_seconds", nil))
		assert.Positive(t, metricValue(t, "orakl_delegator_last_sign_success_timestamp_seconds", nil))
	})

	t.Run("auth_failed", func(t *testing.T) {
		tx, _ := signedSenderTx(t, contract, call, chainID)
		assertCounted(t, metrics.ResultAuthFailed, "", func() {
			code, _ := signRequest(t, app, tx, "")
			assert.Equal(t, http.StatusUnauthorized, code)
		})
		assertCounted(t, metrics.ResultAuthFailed, "", func() {
			code, _ := signRequest(t, app, tx, token+"x")
			assert.Equal(t, http.StatusUnauthorized, code)
		})
	})

	t.Run("not_allowlisted", func(t *testing.T) {
		assertCounted(t, metrics.ResultNotAllowlisted, "node-a", func() {
			tx, _ := signedSenderTx(t, contract, []byte{0xde, 0xad, 0xbe, 0xef}, chainID)
			code, _ := signRequest(t, app, tx, token)
			assert.Equal(t, http.StatusForbidden, code)
		})
		assertCounted(t, metrics.ResultNotAllowlisted, "node-a", func() {
			tx, _ := signedSenderTx(t, common.HexToAddress("0x1234"), call, chainID)
			code, _ := signRequest(t, app, tx, token)
			assert.Equal(t, http.StatusForbidden, code)
		})
	})

	t.Run("invalid_tx", func(t *testing.T) {
		valid, _ := signedSenderTx(t, contract, call, chainID)
		noSelector, _ := signedSenderTx(t, contract, []byte{0x01}, chainID)
		legacy := types.NewTransaction(0, contract, big.NewInt(0), 90000, big.NewInt(1), call)
		cases := map[string]string{
			"malformed body":   `{`,
			"rawTx not hex":    `{"rawTx":"0xzz"}`,
			"missing fields":   `{"rawTx":"0x` + TxToHash(valid) + `"}`,
			"no selector":      payloadJSON(t, noSelector),
			"legacy (no from)": payloadJSON(t, legacy),
		}
		for name, body := range cases {
			assertCounted(t, metrics.ResultInvalidTx, "node-a", func() {
				code := rawSignRequest(t, app, body, token)
				assert.Equal(t, http.StatusBadRequest, code, name)
			})
		}
	})

	t.Run("error: no fee payer key", func(t *testing.T) {
		utils.UpdateFeePayer("")
		t.Cleanup(func() { utils.UpdateFeePayer(feePayerPK) })
		assertCounted(t, metrics.ResultError, "node-a", func() {
			tx, _ := signedSenderTx(t, contract, call, chainID)
			code, _ := signRequest(t, app, tx, token)
			assert.Equal(t, http.StatusInternalServerError, code)
		})
	})

	t.Run("other /sign routes are not counted and still need a token", func(t *testing.T) {
		assertCounted(t, "", "", func() {
			for _, path := range []string{"/api/v1/sign/feePayer", "/api/v1/sign/initialize"} {
				req, err := http.NewRequest(http.MethodGet, path, nil)
				require.NoError(t, err)
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				resp.Body.Close()
				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, path)
			}
		})
	})

	t.Run("error: token store failure", func(t *testing.T) {
		lookups := storeErrors(t, metrics.StoreOpTokenLookup)
		require.NoError(t, appConfig.Store.Close())
		assertCounted(t, metrics.ResultError, "", func() {
			tx, _ := signedSenderTx(t, contract, call, chainID)
			code, _ := signRequest(t, app, tx, token)
			assert.Equal(t, http.StatusInternalServerError, code)
		})
		assert.Equal(t, lookups+1, storeErrors(t, metrics.StoreOpTokenLookup))
	})
}

func payloadJSON(t *testing.T, tx *types.Transaction) string {
	t.Helper()
	return `{"from":"0x1","to":"0x1","input":"0x","gas":"1","value":"0","chainId":"1","gasPrice":"1","nonce":"1","v":"1","r":"1","s":"1","rawTx":"0x` + TxToHash(tx) + `"}`
}

func TestAllowlistRejection(t *testing.T) {
	contract := common.HexToAddress("0x1234")
	tx := feeDelegatedCall(t, contract, []byte{1, 2, 3, 4})
	cases := []struct {
		err    error
		result string
		code   int
	}{
		{notAllowlistedError("not approved contract address"), metrics.ResultNotAllowlisted, http.StatusForbidden},
		{errors.New("database is locked"), metrics.ResultError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		app := fiber.New(fiber.Config{ErrorHandler: utils.CustomErrorHandler})
		app.Post("/", observeSign, func(c *fiber.Ctx) error {
			return allowlistRejection(c, tx, "0x1234", "0x01020304", metrics.StoreOpContractLookup, tc.err)
		})
		lookups := storeErrors(t, metrics.StoreOpContractLookup)
		assertCounted(t, tc.result, "", func() {
			resp, err := app.Test(httptestRequest(t), -1)
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, tc.code, resp.StatusCode)
		})
		wantLookups := lookups
		if tc.result == metrics.ResultError {
			wantLookups++
		}
		assert.Equal(t, wantLookups, storeErrors(t, metrics.StoreOpContractLookup))
	}
}

func httptestRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, err)
	return req
}
