package sign

import (
	"math/big"
	"testing"

	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/assert"
)

func feeDelegatedCall(t *testing.T, to common.Address, data []byte) *types.Transaction {
	t.Helper()
	tx, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    uint64(0),
		types.TxValueKeyGasPrice: big.NewInt(1),
		types.TxValueKeyGasLimit: uint64(90000),
		types.TxValueKeyTo:       to,
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     common.HexToAddress("0x1"),
		types.TxValueKeyData:     data,
		types.TxValueKeyFeePayer: common.HexToAddress("0x2"),
	})
	assert.NoError(t, err)
	return tx
}

func TestTxTarget(t *testing.T) {
	to := common.HexToAddress("0xCB56B163E545A3870CA04C6AE2401F2405FB29A9")

	addr, selector, err := txTarget(feeDelegatedCall(t, to, common.FromHex("0x202ee0ed00000000")))
	assert.NoError(t, err)
	assert.Equal(t, "0xcb56b163e545a3870ca04c6ae2401f2405fb29a9", addr)
	assert.Equal(t, "0x202ee0ed", selector)

	_, _, err = txTarget(feeDelegatedCall(t, to, common.FromHex("0x202ee0")))
	assert.Error(t, err, "data shorter than a selector")

	_, _, err = txTarget(types.NewContractCreation(0, big.NewInt(0), 90000, big.NewInt(1), common.FromHex("0x6080")))
	assert.Error(t, err, "contract creation")
}
