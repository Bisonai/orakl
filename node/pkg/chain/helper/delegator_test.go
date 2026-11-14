package helper

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
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
