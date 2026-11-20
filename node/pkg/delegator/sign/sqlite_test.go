package sign

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/delegator/store"
	"bisonai.com/miko/node/pkg/delegator/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const submitSignature = "submit(bytes32[],int256[],uint256[],bytes[])"

func signedSenderTx(t *testing.T, to common.Address, data []byte, chainID *big.Int) (*types.Transaction, common.Address) {
	t.Helper()
	senderKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(senderKey.PublicKey)
	tx, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    uint64(7),
		types.TxValueKeyGasPrice: big.NewInt(25),
		types.TxValueKeyGasLimit: uint64(90000),
		types.TxValueKeyTo:       to,
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     sender,
		types.TxValueKeyData:     data,
		types.TxValueKeyFeePayer: common.HexToAddress("0x2"),
	})
	require.NoError(t, err)
	require.NoError(t, tx.SignWithKeys(types.NewEIP155Signer(chainID), []*ecdsa.PrivateKey{senderKey}))
	return tx, sender
}

func signRequest(t *testing.T, app *fiber.App, tx *types.Transaction, token string) (int, string) {
	t.Helper()
	payload := map[string]string{
		"from": "0x1", "to": "0x1", "input": "0x", "gas": "1", "value": "0", "chainId": "1",
		"gasPrice": "1", "nonce": "1", "v": "1", "r": "1", "s": "1", "rawTx": "0x" + TxToHash(tx),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/sign/v2", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw)
}

func TestSignV2SQLite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	contract := common.HexToAddress("0xCB56B163E545A3870CA04C6AE2401F2405FB29A9")

	s, err := store.Open(ctx, dir, 0, true)
	require.NoError(t, err)
	_, err = s.Upsert(ctx, contract.Hex(), submitSignature, "SubmissionProxy submit")
	require.NoError(t, err)
	token, err := s.AddToken(ctx, "node-a")
	require.NoError(t, err)
	require.NoError(t, s.Close())

	feePayerKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	t.Setenv("DATABASE_URL", "")
	t.Setenv(store.EnvDir, dir)
	t.Setenv("DELEGATOR_FEEPAYER_PK", hex.EncodeToString(crypto.FromECDSA(feePayerKey)))

	appConfig, err := utils.Setup("test")
	require.NoError(t, err)
	require.NotNil(t, appConfig.Store)
	defer appConfig.Store.Close()
	SigningRoutes(appConfig.App.Group("/api/v1"))

	chainID := big.NewInt(1001)
	selector := crypto.Keccak256([]byte(submitSignature))[:4]

	t.Run("allowed function is signed and recorded", func(t *testing.T) {
		tx, sender := signedSenderTx(t, contract, append(selector, 0x01), chainID)
		code, body := signRequest(t, appConfig.App, tx, token)
		require.Equal(t, http.StatusOK, code, body)

		var result SignModel
		require.NoError(t, json.Unmarshal([]byte(body), &result))
		require.NotNil(t, result.SignedRawTx)
		signed, err := HashToTx(*result.SignedRawTx)
		require.NoError(t, err)
		feePayer, err := types.SenderFeePayer(types.NewEIP155Signer(chainID), signed)
		require.NoError(t, err)
		assert.Equal(t, crypto.PubkeyToAddress(feePayerKey.PublicKey), feePayer)

		files, err := appConfig.Store.TransactionFiles()
		require.NoError(t, err)
		require.Len(t, files, 1)
		db, err := sql.Open("sqlite", filepath.Join(dir, files[0]))
		require.NoError(t, err)
		defer db.Close()
		var gotSender, gotSelector, gotHash, gotToken string
		var gotNonce int64
		require.NoError(t, db.QueryRow("SELECT sender, selector, nonce, tx_hash, token FROM transactions").Scan(&gotSender, &gotSelector, &gotNonce, &gotHash, &gotToken))
		assert.Equal(t, "node-a", gotToken)
		assert.Equal(t, bytes.ToLower([]byte(sender.Hex())), []byte(gotSender))
		assert.Equal(t, "0x"+hex.EncodeToString(selector), gotSelector)
		assert.Equal(t, int64(7), gotNonce)
		assert.Equal(t, signed.Hash().Hex(), gotHash)
	})

	t.Run("store stays open across requests", func(t *testing.T) {
		tx, _ := signedSenderTx(t, contract, append(selector, 0x02), chainID)
		code, body := signRequest(t, appConfig.App, tx, token)
		require.Equal(t, http.StatusOK, code, body)

		files, err := appConfig.Store.TransactionFiles()
		require.NoError(t, err)
		db, err := sql.Open("sqlite", filepath.Join(dir, files[0]))
		require.NoError(t, err)
		defer db.Close()
		var n int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&n))
		assert.Equal(t, 2, n)

		ok, err := appConfig.Store.ContractAllowed(context.Background(), strings.ToLower(contract.Hex()))
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("missing or wrong token is refused", func(t *testing.T) {
		tx, _ := signedSenderTx(t, contract, append(selector, 0x03), chainID)
		code, _ := signRequest(t, appConfig.App, tx, "")
		assert.Equal(t, http.StatusUnauthorized, code)
		code, _ = signRequest(t, appConfig.App, tx, token+"x")
		assert.Equal(t, http.StatusUnauthorized, code)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/sign/feePayer", nil)
		require.NoError(t, err)
		resp, err := appConfig.App.Test(req, -1)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "every /sign route needs a token")
	})

	t.Run("revoked token is refused", func(t *testing.T) {
		other, err := appConfig.Store.AddToken(context.Background(), "node-b")
		require.NoError(t, err)
		tx, _ := signedSenderTx(t, contract, append(selector, 0x04), chainID)
		code, body := signRequest(t, appConfig.App, tx, other)
		require.Equal(t, http.StatusOK, code, body)

		removed, err := appConfig.Store.RemoveToken(context.Background(), "node-b")
		require.NoError(t, err)
		require.True(t, removed)
		code, _ = signRequest(t, appConfig.App, tx, other)
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("other function is refused", func(t *testing.T) {
		tx, _ := signedSenderTx(t, contract, []byte{0xde, 0xad, 0xbe, 0xef}, chainID)
		code, _ := signRequest(t, appConfig.App, tx, token)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("other contract is refused", func(t *testing.T) {
		tx, _ := signedSenderTx(t, common.HexToAddress("0x1234"), append(selector, 0x01), chainID)
		code, _ := signRequest(t, appConfig.App, tx, token)
		assert.Equal(t, http.StatusForbidden, code)
	})

	post := func(t *testing.T, path string, payload map[string]string) int {
		t.Helper()
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := appConfig.App.Test(req, -1)
		require.NoError(t, err)
		resp.Body.Close()
		return resp.StatusCode
	}

	t.Run("request to field naming an allowed contract does not matter", func(t *testing.T) {
		// RawTx is what gets signed; its target is not allowlisted
		tx, _ := signedSenderTx(t, common.HexToAddress("0x1234"), append(selector, 0x01), chainID)
		code := post(t, "/api/v1/sign/v2", map[string]string{
			"from": "0x1", "to": strings.ToLower(contract.Hex()), "input": "0x" + hex.EncodeToString(append(selector, 0x01)),
			"gas": "1", "value": "0", "chainId": "1", "gasPrice": "1", "nonce": "1", "v": "1", "r": "1", "s": "1",
			"rawTx": "0x" + TxToHash(tx),
		})
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("legacy signing endpoints are not served", func(t *testing.T) {
		tx, _ := signedSenderTx(t, contract, append(selector, 0x05), chainID)
		payload := map[string]string{"rawTx": "0x" + TxToHash(tx)}
		for _, path := range []string{"/api/v1/sign", "/api/v1/sign/volatile"} {
			assert.NotEqual(t, http.StatusOK, post(t, path, payload), path)
		}
	})
}

func TestSetupSQLiteRequiresAllowlist(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv(store.EnvDir, t.TempDir())
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	_, err := utils.Setup("test")
	assert.ErrorContains(t, err, "config.db")
}

func TestSetupRequiresSQLiteDir(t *testing.T) {
	t.Setenv(store.EnvDir, "")
	t.Setenv("DATABASE_URL", "postgres://unused")
	_, err := utils.Setup("test")
	assert.ErrorContains(t, err, store.EnvDir+" is required")
}

func TestSetupIgnoresDatabaseURL(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(context.Background(), dir, 0, true)
	require.NoError(t, err)
	token, err := s.AddToken(context.Background(), "node-a")
	require.NoError(t, err)
	require.NoError(t, s.Close())

	t.Setenv("DATABASE_URL", "postgres://unused")
	t.Setenv(store.EnvDir, dir)
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	appConfig, err := utils.Setup("test")
	require.NoError(t, err)
	require.NotNil(t, appConfig.Store)
	defer appConfig.Store.Close()
	// no fee payer: Setup started the retry loop, which stops on shutdown
	t.Cleanup(func() { _ = appConfig.App.Shutdown() })
	SigningRoutes(appConfig.App.Group("/api/v1"))

	// sign routes still require a token with DATABASE_URL set
	tx, _ := signedSenderTx(t, common.HexToAddress("0x1234"), []byte{1, 2, 3, 4}, big.NewInt(1001))
	code, _ := signRequest(t, appConfig.App, tx, "")
	assert.Equal(t, http.StatusUnauthorized, code)
	code, _ = signRequest(t, appConfig.App, tx, token)
	assert.Equal(t, http.StatusForbidden, code)
}

func TestInitializeAndFeePayerSQLite(t *testing.T) {
	// the fee payer is package-global: restore it for later tests
	prev := utils.CurrentFeePayer()
	t.Cleanup(func() { utils.UpdateFeePayer(prev) })

	dir := t.TempDir()
	s, err := store.Open(context.Background(), dir, 0, true)
	require.NoError(t, err)
	token, err := s.AddToken(context.Background(), "node-a")
	require.NoError(t, err)
	require.NoError(t, s.Close())

	const sourceKey = "6014d3aa8be8980fd90146d10176e7ef632bdba96279e8bbe55421e79a979a2e"
	t.Setenv(store.EnvDir, dir)
	t.Setenv("DELEGATOR_FEEPAYER_PK", sourceKey)
	appConfig, err := utils.Setup("test")
	require.NoError(t, err)
	defer appConfig.Store.Close()
	SigningRoutes(appConfig.App.Group("/api/v1"))

	get := func(path string) (int, string) {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := appConfig.App.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(raw)
	}

	code, body := get("/api/v1/sign/feePayer")
	require.Equal(t, http.StatusOK, code, body)
	assert.Equal(t, `"0xda2E0E7089a479ef4A75a8c6Cc78426B9270EC08"`, body)

	const overrideKey = "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	utils.UpdateFeePayer(overrideKey)

	// a key in the request is refused and the current key stays in place
	code, _ = get("/api/v1/sign/initialize?feePayerPrivateKey=0x" + sourceKey)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, overrideKey, utils.CurrentFeePayer())

	// without a key it reloads from the configured source
	code, body = get("/api/v1/sign/initialize")
	require.Equal(t, http.StatusOK, code, body)
	assert.Equal(t, sourceKey, utils.CurrentFeePayer())
}
