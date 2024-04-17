package reporter

import (
	"context"
	"testing"
	"time"

	chain_utils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/klaytn/klaytn/common"
	"github.com/stretchr/testify/assert"
)

func TestCheckForNonWhitelistedSigners(t *testing.T) {
	whitelist := []common.Address{
		common.HexToAddress("0x1234567890abcdef"),
		common.HexToAddress("0xabcdef1234567890"),
	}

	t.Run("All signers whitelisted", func(t *testing.T) {
		signers := []common.Address{
			common.HexToAddress("0x1234567890abcdef"),
			common.HexToAddress("0xabcdef1234567890"),
		}

		err := CheckForNonWhitelistedSigners(signers, whitelist)

		assert.NoError(t, err)
	})

	t.Run("One non-whitelisted signer", func(t *testing.T) {
		signers := []common.Address{
			common.HexToAddress("0x1234567890abcdef"),
			common.HexToAddress("0xabcdef1234567890"),
			common.HexToAddress("0xdeadbeefdeadbeef"),
		}

		err := CheckForNonWhitelistedSigners(signers, whitelist)

		assert.EqualError(t, err, "non-whitelisted signer")
	})

	t.Run("Empty signers", func(t *testing.T) {
		signers := []common.Address{}

		err := CheckForNonWhitelistedSigners(signers, whitelist)

		assert.NoError(t, err)
	})
}

func TestOrderProof(t *testing.T) {
	t.Run("Valid proofs", func(t *testing.T) {
		signerMap := map[common.Address][]byte{
			common.HexToAddress("0x1234567890abcdef"): []byte("proof1"),
			common.HexToAddress("0xabcdef1234567890"): []byte("proof2"),
			common.HexToAddress("0xdeadbeefdeadbeef"): []byte("proof3"),
		}

		whitelist := []common.Address{
			common.HexToAddress("0x1234567890abcdef"),
			common.HexToAddress("0xabcdef1234567890"),
		}

		expectedProof := []byte("proof1proof2")
		proof, err := OrderProof(signerMap, whitelist)

		assert.NoError(t, err)
		assert.Equal(t, expectedProof, proof)
	})

	t.Run("Order should change", func(t *testing.T) {
		signerMap := map[common.Address][]byte{
			common.HexToAddress("0x1234567890abcdef"): []byte("proof1"),
			common.HexToAddress("0xabcdef1234567890"): []byte("proof2"),
			common.HexToAddress("0xdeadbeefdeadbeef"): []byte("proof3"),
		}

		whitelist := []common.Address{
			common.HexToAddress("0xabcdef1234567890"),
			common.HexToAddress("0x1234567890abcdef"),
		}

		expectedProof := []byte("proof2proof1")
		proof, err := OrderProof(signerMap, whitelist)

		assert.NoError(t, err)
		assert.Equal(t, expectedProof, proof)
	})

	t.Run("No valid proofs", func(t *testing.T) {
		signerMap := map[common.Address][]byte{
			common.HexToAddress("0xdeadbeefdeadbeef"): []byte("proof3"),
		}

		whitelist := []common.Address{
			common.HexToAddress("0x1234567890abcdef"),
			common.HexToAddress("0xabcdef1234567890"),
		}

		proof, err := OrderProof(signerMap, whitelist)

		assert.EqualError(t, err, "no valid proofs")
		assert.Nil(t, proof)
	})
}
func TestGetSignerMap(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x1234567890abcdef"),
		common.HexToAddress("0xabcdef1234567890"),
		common.HexToAddress("0xdeadbeefdeadbeef"),
	}

	proofChunks := [][]byte{
		[]byte("proof1"),
		[]byte("proof2"),
		[]byte("proof3"),
	}

	expectedSignerMap := map[common.Address][]byte{
		common.HexToAddress("0x1234567890abcdef"): []byte("proof1"),
		common.HexToAddress("0xabcdef1234567890"): []byte("proof2"),
		common.HexToAddress("0xdeadbeefdeadbeef"): []byte("proof3"),
	}

	signerMap := GetSignerMap(signers, proofChunks)

	assert.Equal(t, expectedSignerMap, signerMap)
}

func TestGetSignerListFromProofs(t *testing.T) {
	testValue := int64(10)
	testTimestamp := time.Now().Unix()

	hash := chain_utils.Value2HashForSign(testValue, testTimestamp)
	test_pk_0 := "737ea08c90c582aafdd7644ec492ee685df711df1ca055fd351938a493058217"
	test_pk_1 := "c2235dcc40306325e1e060b066edb728a1734a377a9648461526101e5365ac56"
	pk_0, err := chain_utils.StringToPk(test_pk_0)
	if err != nil {
		t.Fatalf("Failed to convert string to pk: %v", err)
	}
	pk_1, err := chain_utils.StringToPk(test_pk_1)
	if err != nil {
		t.Fatalf("Failed to convert string to pk: %v", err)
	}

	sig_0, err := chain_utils.MakeValueSignature(testValue, testTimestamp, pk_0)
	if err != nil {
		t.Fatalf("Failed to make value signature: %v", err)
	}
	sig_1, err := chain_utils.MakeValueSignature(testValue, testTimestamp, pk_1)
	if err != nil {
		t.Fatalf("Failed to make value signature: %v", err)
	}

	proofChunks := [][]byte{
		sig_0,
		sig_1,
	}

	expectedSigners := []common.Address{
		common.HexToAddress("0x2138824ef8741add09E8680F968e1d5D0AC155E0"),
		common.HexToAddress("0xd7b29E19c08d412d9c3c96D00cC53609F313D4E9"),
	}

	signers, err := GetSignerListFromProofs(hash, proofChunks)

	assert.NoError(t, err)
	assert.Equal(t, expectedSigners, signers)
}

func TestSplitProofToChunk(t *testing.T) {
	t.Run("Empty proof", func(t *testing.T) {
		proof := []byte{}
		chunks, err := SplitProofToChunk(proof)

		assert.Nil(t, chunks)
		assert.EqualError(t, err, "empty proof")
	})

	t.Run("Invalid proof length", func(t *testing.T) {
		proof := []byte("invalidproof")
		chunks, err := SplitProofToChunk(proof)

		assert.Nil(t, chunks)
		assert.EqualError(t, err, "invalid proof length")
	})

	t.Run("Valid proof", func(t *testing.T) {
		proof := []byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalidvalidproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid")
		expectedChunks := [][]byte{
			[]byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid"),
			[]byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid"),
		}
		chunks, err := SplitProofToChunk(proof)

		assert.Equal(t, expectedChunks, chunks)
		assert.NoError(t, err)
	})
}

func TestRemoveDuplicateProof(t *testing.T) {
	t.Run("Empty proof", func(t *testing.T) {
		proof := []byte{}
		expectedResult := []byte{}
		result := RemoveDuplicateProof(proof)

		assert.Equal(t, expectedResult, result)
	})

	t.Run("No duplicate proofs", func(t *testing.T) {
		proof := []byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid")
		expectedResult := []byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid")
		result := RemoveDuplicateProof(proof)

		assert.Equal(t, expectedResult, result)
	})

	t.Run("Duplicate proofs", func(t *testing.T) {
		proof := []byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalidvalidproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid")
		expectedResult := []byte("validproofvalidproofvalidproofvalidproofvalidproofvalidproofvalid")
		result := RemoveDuplicateProof(proof)

		assert.Equal(t, expectedResult, result)
	})
}

func TestUpsertProofs(t *testing.T) {
	ctx := context.Background()

	aggregates := []GlobalAggregate{
		{Name: "aggregate1", Round: 1},
		{Name: "aggregate2", Round: 2},
		{Name: "aggregate3", Round: 3},
	}

	proofMap := map[string][]byte{
		"aggregate1": []byte("proof1"),
		"aggregate2": []byte("proof2"),
	}

	expectedUpsertRows := [][]any{
		{"aggregate1", int64(1), []byte("proof1")},
		{"aggregate2", int64(2), []byte("proof2")},
	}
	err := UpsertProofs(ctx, aggregates, proofMap)
	if err != nil {
		t.Fatalf("UpsertProofs failed: %v", err)
	}
	result, err := db.QueryRows[PgsqlProof](ctx, "SELECT * FROM proofs WHERE name IN ('aggregate1', 'aggregate2')", nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	for i, p := range result {
		assert.Equal(t, expectedUpsertRows[i], []any{p.Name, p.Round, p.Proof})
	}

	proofMap = map[string][]byte{
		"aggregate1": []byte("proof3"),
		"aggregate2": []byte("proof4"),
	}
	expectedUpsertRows = [][]any{
		{"aggregate1", int64(1), []byte("proof3")},
		{"aggregate2", int64(2), []byte("proof4")},
	}
	err = UpsertProofs(ctx, aggregates, proofMap)
	if err != nil {
		t.Fatalf("UpsertProofs failed: %v", err)
	}
	result, err = db.QueryRows[PgsqlProof](ctx, "SELECT * FROM proofs WHERE name IN ('aggregate1', 'aggregate2')", nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	for i, p := range result {
		assert.Equal(t, expectedUpsertRows[i], []any{p.Name, p.Round, p.Proof})
	}

	assert.NoError(t, err)

	db.QueryWithoutResult(ctx, "DELETE FROM proofs WHERE name IN ('aggregate1', 'aggregate2')", nil)
}
