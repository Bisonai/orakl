package reporter

import (
	"context"
	"strconv"
	"testing"
	"time"

	chainUtils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
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

		assert.ErrorIs(t, err, errorSentinel.ErrReporterSignerNotWhitelisted)
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

		assert.ErrorIs(t, err, errorSentinel.ErrReporterEmptyValidProofs)
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
	testName := "test"

	hash := chainUtils.Value2HashForSign(testValue, testTimestamp, testName)
	test_pk_0 := "737ea08c90c582aafdd7644ec492ee685df711df1ca055fd351938a493058217"
	test_pk_1 := "c2235dcc40306325e1e060b066edb728a1734a377a9648461526101e5365ac56"
	pk_0, err := chainUtils.StringToPk(test_pk_0)
	if err != nil {
		t.Fatalf("Failed to convert string to pk: %v", err)
	}
	pk_1, err := chainUtils.StringToPk(test_pk_1)
	if err != nil {
		t.Fatalf("Failed to convert string to pk: %v", err)
	}

	sig_0, err := chainUtils.MakeValueSignature(testValue, testTimestamp, testName, pk_0)
	if err != nil {
		t.Fatalf("Failed to make value signature: %v", err)
	}
	sig_1, err := chainUtils.MakeValueSignature(testValue, testTimestamp, testName, pk_1)
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
		assert.ErrorIs(t, err, errorSentinel.ErrReporterEmptyProofParam)
	})

	t.Run("Invalid proof length", func(t *testing.T) {
		proof := []byte("invalidproof")
		chunks, err := SplitProofToChunk(proof)

		assert.Nil(t, chunks)
		assert.ErrorIs(t, err, errorSentinel.ErrReporterInvalidProofLength)
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

func TestUpdateProofs(t *testing.T) {
	ctx := context.Background()
	defer func() {
		err := db.QueryWithoutResult(ctx, "DELETE FROM proofs", nil)
		if err != nil {
			t.Logf("QueryWithoutResult failed: %v", err)
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM configs", nil)
		if err != nil {
			t.Logf("QueryWithoutResult failed: %v", err)
		}
	}()

	tmpConfigs := []Config{}
	for i := 0; i < 3; i++ {
		tmpConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{
			"name":               "test-aggregate-" + strconv.Itoa(i),
			"address":            "0x1234" + strconv.Itoa(i),
			"submit_interval":    TestInterval,
			"fetch_interval":     TestInterval,
			"aggregate_interval": TestInterval})
		if err != nil {
			t.Fatalf("QueryRow failed: %v", err)
		}
		tmpConfigs = append(tmpConfigs, tmpConfig)
	}

	aggregates := []GlobalAggregate{
		{ConfigID: tmpConfigs[0].ID, Round: 1},
		{ConfigID: tmpConfigs[1].ID, Round: 2},
		{ConfigID: tmpConfigs[2].ID, Round: 3},
	}

	proofMap := map[int32][]byte{
		tmpConfigs[0].ID: []byte("proof1"),
		tmpConfigs[1].ID: []byte("proof2"),
	}

	expectedUpsertRows := [][]any{
		{tmpConfigs[0].ID, int32(1), []byte("proof1")},
		{tmpConfigs[1].ID, int32(2), []byte("proof2")},
	}
	err := UpsertProofs(ctx, aggregates, proofMap)
	if err != nil {
		t.Fatalf("UpsertProofs failed: %v", err)
	}
	result, err := db.QueryRows[Proof](ctx, "SELECT * FROM proofs WHERE config_id IN ("+strconv.Itoa(int(tmpConfigs[0].ID))+", "+strconv.Itoa(int(tmpConfigs[1].ID))+")", nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	for i, p := range result {
		assert.Equal(t, expectedUpsertRows[i], []any{p.ConfigID, p.Round, p.Proof})
	}

	proofMap = map[int32][]byte{
		tmpConfigs[0].ID: []byte("proof3"),
		tmpConfigs[1].ID: []byte("proof4"),
	}
	expectedUpsertRows = [][]any{
		{tmpConfigs[0].ID, int32(1), []byte("proof3")},
		{tmpConfigs[1].ID, int32(2), []byte("proof4")},
	}
	err = UpdateProofs(ctx, aggregates, proofMap)
	if err != nil {
		t.Fatalf("UpdateProofs failed: %v", err)
	}
	result, err = db.QueryRows[Proof](ctx, "SELECT * FROM proofs WHERE config_id IN ("+strconv.Itoa(int(tmpConfigs[0].ID))+", "+strconv.Itoa(int(tmpConfigs[1].ID))+")", nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	for i, p := range result {
		assert.Equal(t, expectedUpsertRows[i], []any{p.ConfigID, p.Round, p.Proof})
	}
}
