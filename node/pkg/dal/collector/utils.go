package collector

import (
	"context"
	"errors"

	"bisonai.com/miko/node/pkg/chain/chainreader"

	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func getAllOracles(ctx context.Context, chainReader *chainreader.ChainReader, submissionProxyContractAddr string) ([]kaiacommon.Address, error) {
	rawResult, err := chainReader.ReadContract(ctx, submissionProxyContractAddr, GetAllOracles)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all oracles")
		return nil, err
	}
	rawResultSlice, ok := rawResult.([]interface{})
	if !ok {
		return nil, errors.New("failed to cast result to []interface{} in getAllOracles")
	}

	addresses, ok := rawResultSlice[0].([]kaiacommon.Address)
	if !ok {
		return nil, errors.New("failed to cast first element to []kaiacommon.Address")
	}

	log.Info().Any("addresses", addresses).Msg("loaded oracles")

	return addresses, nil
}

func formatBytesToHex(bytes []byte) string {
	return "0x" + kaiacommon.Bytes2Hex(bytes)
}
