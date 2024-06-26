package websocketchainreader

import (
	"context"
	"math/big"
	"time"

	"bisonai.com/orakl/node/pkg/chain/eth_client"
	"bisonai.com/orakl/node/pkg/chain/utils"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func New(kaiaWssUrl, ethWssUrl string) (*ChainReader, error) {
	if kaiaWssUrl == "" {
		log.Error().Str("Player", "ChainReader").Err(errorSentinel.ErrChainKaiaWebsocketUrlNotFound).Msg("Kaia websocket url not found")
		return nil, errorSentinel.ErrChainKaiaWebsocketUrlNotFound
	}

	if ethWssUrl == "" {
		log.Error().Str("Player", "ChainReader").Err(errorSentinel.ErrChainEthWebsocketUrlNotFound).Msg("Eth websocket url not found")
		return nil, errorSentinel.ErrChainEthWebsocketUrlNotFound
	}

	ethClient, err := eth_client.Dial(ethWssUrl)
	if err != nil {
		log.Error().Str("Player", "ChainReader").Err(err).Msg("Failed to connect to eth websocket")
		return nil, err
	}

	ethChainId, err := ethClient.ChainID(context.Background())
	if err != nil {
		log.Error().Str("Player", "ChainReader").Err(err).Msg("Failed to get eth chain id")
		return nil, err
	}

	kaiaClient, err := client.Dial(kaiaWssUrl)
	if err != nil {
		log.Error().Str("Player", "ChainReader").Err(err).Msg("Failed to connect to kaia websocket")
		return nil, err
	}

	kaiaChainId, err := kaiaClient.ChainID(context.Background())
	if err != nil {
		log.Error().Str("Player", "ChainReader").Err(err).Msg("Failed to get kaia chain id")
		return nil, err
	}

	chainIdToChainType := make(map[string]BlockchainType)
	chainIdToChainType[ethChainId.String()] = Ethereum
	chainIdToChainType[kaiaChainId.String()] = Kaia

	return &ChainReader{
		EthClient:          ethClient,
		KaiaClient:         kaiaClient,
		RetryPeriod:        1 * time.Second,
		ChainIdToChainType: chainIdToChainType,
	}, nil
}

func (c *ChainReader) BlockNumber(ctx context.Context, chainType BlockchainType) (*big.Int, error) {
	websocketClient := c.client(chainType)
	return websocketClient.BlockNumber(ctx)
}

func (c *ChainReader) Subscribe(ctx context.Context, opts ...SubscribeOption) error {
	config := &SubscribeConfig{
		ChainType: Ethereum,
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.Address == "" {
		return errorSentinel.ErrChainWebsocketContractAddressNotfound
	}

	if config.Ch == nil {
		return errorSentinel.ErrChainWebsocketChannelNotfound
	}

	go c.handleSubscription(ctx, config)
	return nil
}

func (c *ChainReader) handleSubscription(ctx context.Context, config *SubscribeConfig) {
	initialTrigger := true
	for {
		blockNumber, err := c.getBlockNumber(ctx, config, initialTrigger)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get block number, retrying")
			if !retryWithContext(ctx, c.RetryPeriod) {
				return
			}
			log.Debug().Err(err).Str("Player", "ChainReader").Msg("Retrying subscription")
			continue
		}

		query := klaytn.FilterQuery{
			FromBlock: blockNumber,
			Addresses: []common.Address{common.HexToAddress(config.Address)},
		}

		logs := make(chan types.Log)
		sub, err := c.client(config.ChainType).SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			log.Error().Err(err).Msg("Failed to subscribe, retrying")
			if !retryWithContext(ctx, c.RetryPeriod) {
				return
			}
			log.Debug().Err(err).Str("Player", "ChainReader").Msg("Retrying subscription")
			continue
		}
		defer sub.Unsubscribe()

		if !processLogs(ctx, sub, logs, config.Ch) {
			if !retryWithContext(ctx, c.RetryPeriod) {
				return
			}
			log.Debug().Err(err).Str("Player", "ChainReader").Msg("Retrying subscription")
			continue
		}

		initialTrigger = false
	}
}

func (c *ChainReader) getBlockNumber(ctx context.Context, config *SubscribeConfig, initialTrigger bool) (*big.Int, error) {
	if initialTrigger && config.BlockNumber != nil {
		return config.BlockNumber, nil
	}

	return c.client(config.ChainType).BlockNumber(ctx)
}

func (c *ChainReader) client(chainType BlockchainType) utils.ClientInterface {
	if chainType == Ethereum {
		return c.EthClient
	}

	return c.KaiaClient
}

func (c *ChainReader) ReadContractOnce(ctx context.Context, chain BlockchainType, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return utils.ReadContract(ctx, c.client(chain), functionString, contractAddressHex, args...)
}

func retryWithContext(ctx context.Context, duration time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(duration):
		return true
	}
}

func processLogs(ctx context.Context, sub klaytn.Subscription, logs <-chan types.Log, ch chan<- types.Log) bool {
	for {
		select {
		case err := <-sub.Err():
			log.Error().Err(err).Msg("Error in subscription")
			return false
		case vLog := <-logs:
			select {
			case ch <- vLog:
				continue
			case <-ctx.Done():
				return false
			}
		}
	}
}
