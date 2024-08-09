package signer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/accounts/abi"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func ExtractExpirationFromContract(ctx context.Context, jsonrpc string, submissionProxy string, signer string) (*time.Time, error) {
	klaytnClient, err := client.Dial(jsonrpc)
	if err != nil {
		return nil, err
	}
	defer klaytnClient.Close()

	readResult, err := ReadContract(ctx, *klaytnClient, "whitelist(address) returns ((uint256, uint256))", submissionProxy, common.HexToAddress(signer))
	if err != nil {
		return nil, err
	}

	values := readResult.([]interface{})
	rawTimestamp := values[1].(*big.Int)
	expirationDate := time.Unix(int64(rawTimestamp.Int64()), 0)
	return &expirationDate, nil
}

func GetSignerAddresses(ctx context.Context, jsonrpc string, submissionProxy string) ([]string, error) {
	klaytnClient, err := client.Dial(jsonrpc)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Klaytn client")
		return nil, err
	}
	defer klaytnClient.Close()

	rawData, err := ReadContract(ctx, *klaytnClient, "function getAllOracles() public view returns (address[] memory)", submissionProxy)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read contract")
		return nil, err
	}

	hexData := rawData.([]interface{})
	hexAddress := hexData[0].([]common.Address)
	addresses := make([]string, len(hexAddress))
	for i, address := range hexAddress {
		addresses[i] = address.Hex()
	}

	return addresses, nil
}

func ReadContract(ctx context.Context, client client.Client, functionString string, contractAddress string, args ...interface{}) (interface{}, error) {
	log.Debug().Msg("Preparing to read contract")
	functionName, inputs, outputs, err := ParseMethodSignature(functionString)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse method signature")
		return nil, err
	}

	abi, err := GenerateViewABI(functionName, inputs, outputs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate ABI")
		return nil, err
	}

	contractAddressHex := common.HexToAddress(contractAddress)
	callData, err := abi.Pack(functionName, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to pack call data")
		return nil, err
	}

	result, err := client.CallContract(ctx, klaytn.CallMsg{
		To:   &contractAddressHex,
		Data: callData,
	}, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call contract")
		return nil, err
	}

	output, err := abi.Unpack(functionName, result)

	if err != nil {
		log.Error().Err(err).Msg("Failed to unpack result")
		return nil, err
	}

	return output, nil
}

// reference: https://github.com/umbracle/ethgo/blob/main/abi/abi.go
var (
	funcRegexpWithReturn    = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)\s*returns\s*\((.*)\)`)
	funcRegexpWithoutReturn = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)`)
)

func ParseMethodSignature(name string) (string, string, string, error) {
	if name == "" {
		return "", "", "", errors.New("empty name param for method signature")
	}

	name = strings.Replace(name, "\n", " ", -1)
	name = strings.Replace(name, "\t", " ", -1)

	name = strings.TrimPrefix(name, "function ")
	name = strings.TrimSpace(name)

	var funcName, inputArgs, outputArgs string

	if strings.Contains(name, "returns") {
		matches := funcRegexpWithReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", errors.New("failed to find method signature match")
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
		outputArgs = strings.TrimSpace(matches[0][4])
	} else {
		matches := funcRegexpWithoutReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", errors.New("failed to find method signature match")
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
	}

	return funcName, inputArgs, outputArgs, nil
}

func GenerateViewABI(functionName string, inputs string, outputs string) (*abi.ABI, error) {
	return generateABI(functionName, inputs, outputs, "view", false)
}

func MakeAbiFuncAttribute(args string) string {
	splittedArgs := strings.Split(args, ",")
	if len(splittedArgs) == 0 || splittedArgs[0] == "" {
		return ""
	}

	var parts []string
	for _, arg := range splittedArgs {
		arg = strings.TrimSpace(arg)
		part := strings.Split(arg, " ")

		if len(part) < 2 {
			parts = append(parts, fmt.Sprintf(`{"type":"%s"}`, part[0]))
		} else {
			parts = append(parts, fmt.Sprintf(`{"type":"%s","name":"%s"}`, part[0], part[len(part)-1]))
		}
	}
	return strings.Join(parts, ",\n")
}

func generateABI(functionName string, inputs string, outputs string, stateMutability string, payable bool) (*abi.ABI, error) {
	if functionName == "" {
		return nil, errors.New("empty name param for method signature")
	}

	inputArgs := MakeAbiFuncAttribute(inputs)
	outputArgs := MakeAbiFuncAttribute(outputs)

	json := fmt.Sprintf(`[
		{
			"constant": false,
			"inputs": [%s],
			"name": "%s",
			"outputs": [%s],
			"payable": %t,
			"stateMutability": "%s",
			"type": "function"
		}
	]
	`, inputArgs, functionName, outputArgs, payable, stateMutability)

	parsedABI, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse abi")
		return nil, err
	}

	return &parsedABI, nil
}
