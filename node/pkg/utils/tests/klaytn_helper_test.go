package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewTxHelper(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	txHelper.Close()
}

func TestNextReporter(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	reporter := txHelper.NextReporter()
	if reporter == "" {
		t.Errorf("Unexpected reporter: %v", reporter)
	}
}

func TestMakeDirectTx(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	_, err = txHelper.MakeDirectTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMakeFeeDelegatedTx(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	_, err = txHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTxToHashToTx(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	rawTx, err := txHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := utils.TxToHash(rawTx)
	assert.NotEqual(t, hash, "")

	tx, err := utils.HashToTx(hash)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, tx.Equal(rawTx), true)
}

func TestGenerateABI(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	abi, err := utils.GenerateABI("increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, abi, nil)
}

func TestSubmitRawTxString(t *testing.T) {
	ctx := context.Background()
	txHelper, err := utils.NewTxHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer txHelper.Close()

	rawTx, err := txHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	signedTx, err := txHelper.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rawTxString := utils.TxToHash(signedTx)
	err = txHelper.SubmitRawTxString(ctx, rawTxString)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
