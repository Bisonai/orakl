package websocketchainreader

import (
	"testing"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/kaiachain/kaia/client"
	"github.com/stretchr/testify/assert"
)

type stubClient struct {
	utils.ClientInterface
	closed bool
}

func (s *stubClient) Close() { s.closed = true }

func TestCloseSkipsUnsetClients(t *testing.T) {
	eth := &stubClient{}
	kaia := &stubClient{}
	c := &ChainReader{
		EthClient:      eth,
		KaiaClient:     kaia,
		BaseClient:     (*eth_client.EthClient)(nil), // optional chain, unset
		ArbitrumClient: (*eth_client.EthClient)(nil),
		BscClient:      (*client.Client)(nil),
	}
	assert.NotPanics(t, c.Close)
	assert.True(t, eth.closed)
	assert.True(t, kaia.closed)
}
