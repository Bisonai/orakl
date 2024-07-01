package collector

import (
	"context"
	"errors"
	"time"

	chainutils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"
	"bisonai.com/orakl/node/pkg/reporter"

	"github.com/klaytn/klaytn/blockchain/types"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func getAllOracles(ctx context.Context, chainReader *websocketchainreader.ChainReader, submissionProxyContractAddr string) ([]klaytncommon.Address, error) {
	rawResult, err := chainReader.ReadContractOnce(ctx, websocketchainreader.Kaia, submissionProxyContractAddr, GetAllOracles)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all oracles")
		return nil, err
	}
	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to cast result to []interface{}")
	}

	return rawResultSlice[0].([]klaytncommon.Address), nil
}

func subscribeAddOracleEvent(ctx context.Context, chainReader *websocketchainreader.ChainReader, submissionProxyContractAddr string, isUpdated chan any) error {
	logChannel := make(chan types.Log)
	err := chainReader.Subscribe(
		ctx,
		websocketchainreader.WithAddress(submissionProxyContractAddr),
		websocketchainreader.WithChainType(websocketchainreader.Kaia),
		websocketchainreader.WithChannel(logChannel),
	)
	if err != nil {
		return err
	}

	eventName, input, _, eventParseErr := chainutils.ParseMethodSignature(OracleAdded)
	if eventParseErr != nil {
		return eventParseErr
	}

	oracleAddedEventABI, err := chainutils.GenerateEventABI(eventName, input)
	if err != nil {
		return err
	}

	for eventLog := range logChannel {
		result, err := oracleAddedEventABI.Unpack(eventName, eventLog.Data)
		if err != nil {
			continue
		}

		_, ok := result[0].(klaytncommon.Address)
		if !ok {
			continue
		}

		isUpdated <- true
	}

	return nil
}

func orderProof(ctx context.Context, proof []byte, value int64, timestamp time.Time, symbol string, cachedWhitelist []klaytncommon.Address) ([]byte, error) {
	proof = reporter.RemoveDuplicateProof(proof)
	hash := chainutils.Value2HashForSign(value, timestamp.Unix(), symbol)
	proofChunks, err := reporter.SplitProofToChunk(proof)
	if err != nil {
		return nil, err
	}

	signers, err := reporter.GetSignerListFromProofs(hash, proofChunks)
	if err != nil {
		return nil, err
	}

	err = reporter.CheckForNonWhitelistedSigners(signers, cachedWhitelist)
	if err != nil {
		return nil, err
	}

	signerMap := reporter.GetSignerMap(signers, proofChunks)
	return reporter.OrderProof(signerMap, cachedWhitelist)
}
