package packer

import (
	"math/big"

	"bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/dal/api"
	"github.com/klaytn/klaytn/accounts/abi"
	klaytncommon "github.com/klaytn/klaytn/common"
)

const (
	Submit       = "submit(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
	SubmitStrict = "submitStrict(bytes32[] calldata _feedHashes, int256[] calldata _answers, uint256[] calldata _timestamps, bytes[] calldata _proofs)"
)

var submitAbi *abi.ABI
var submitStrictAbi *abi.ABI

func init() {
	var err error
	submitAbi, err = makeABI(Submit)
	if err != nil {
		panic(err)
	}
	submitStrictAbi, err = makeABI(SubmitStrict)
	if err != nil {
		panic(err)
	}
}

func makeABI(functionString string) (*abi.ABI, error) {
	functionName, inputs, outputs, err := utils.ParseMethodSignature(functionString)
	if err != nil {
		return nil, err
	}
	return utils.GenerateCallABI(functionName, inputs, outputs)

}

func pack(bulk api.BulkResponse) (string, string, error) {
	feedHashes := [][32]byte{}
	for _, feedHash := range bulk.FeedHashes {
		feedHashBytes := klaytncommon.Hex2Bytes(feedHash)
		feedHash := [32]byte{}
		copy(feedHash[:], feedHashBytes)
		feedHashes = append(feedHashes, feedHash)
	}
	values := []*big.Int{}
	for _, value := range bulk.Values {
		var submissionVal big.Int
		submissionVal.SetString(value, 10)
		values = append(values, &submissionVal)
	}
	timestamps := []*big.Int{}
	for _, aggregateTimestamp := range bulk.AggregateTimes {
		var timestamp big.Int
		timestamp.SetString(aggregateTimestamp, 10)

		timestamps = append(timestamps, &timestamp)
	}
	proofs := [][]byte{}
	for _, proof := range bulk.Proofs {
		proofs = append(proofs, klaytncommon.Hex2Bytes(proof))
	}

	packed, err := submitAbi.Pack("submit", feedHashes, values, timestamps, proofs)
	if err != nil {
		return "", "", err
	}
	packedStrict, err := submitStrictAbi.Pack("submitStrict", feedHashes, values, timestamps, proofs)
	if err != nil {
		return "", "", err
	}
	return klaytncommon.Bytes2Hex(packed), klaytncommon.Bytes2Hex(packedStrict), nil

}
