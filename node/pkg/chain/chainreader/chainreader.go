package chainreader

import (
	"context"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/utils"
)

// use eth_client for blockchain interaction

type ChainReader struct {
	client utils.ClientInterface
}

func NewChainReader(jsonRpcUrl string) (*ChainReader, error) {
	client, err := eth_client.Dial(jsonRpcUrl)
	if err != nil {
		return nil, err
	}

	return &ChainReader{
		client: client,
	}, nil
}

func (c *ChainReader) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return utils.ReadContract(ctx, c.client, functionString, contractAddressHex, args...)
}
