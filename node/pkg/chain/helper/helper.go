package helper

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"os"
	"strings"

	"bisonai.com/orakl/node/pkg/chain/noncemanager"
	"bisonai.com/orakl/node/pkg/chain/utils"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/secrets"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/rs/zerolog/log"
)

func setProviderAndReporter(config *ChainHelperConfig, blockchainType BlockchainType) error {
	switch blockchainType {
	case Kaia:
		if config.ProviderUrl == "" {
			config.ProviderUrl = os.Getenv(KaiaProviderUrl)
			if config.ProviderUrl == "" {
				log.Error().Msg("provider url not set")
				return errorSentinel.ErrChainProviderUrlNotFound
			}
		}

		if config.ReporterPk == "" {
			config.ReporterPk = secrets.GetSecret(KaiaReporterPk)
			if config.ReporterPk == "" {
				log.Warn().Msg("reporter pk not set")
			}
		}
	case Ethereum:
		if config.ProviderUrl == "" {
			config.ProviderUrl = os.Getenv(EthProviderUrl)
			if config.ProviderUrl == "" {
				log.Error().Msg("provider url not set")
				return errorSentinel.ErrChainProviderUrlNotFound
			}
		}

		if config.ReporterPk == "" {
			config.ReporterPk = secrets.GetSecret(EthReporterPk)
			if config.ReporterPk == "" {
				log.Warn().Msg("reporter pk not set")
			}
		}
	default:
		return errorSentinel.ErrChainReporterUnsupportedChain
	}

	return nil
}

func NewChainHelper(ctx context.Context, opts ...ChainHelperOption) (*ChainHelper, error) {
	config := &ChainHelperConfig{
		BlockchainType:            Kaia,
		UseAdditionalProviderUrls: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	err := setProviderAndReporter(config, config.BlockchainType)
	if err != nil {
		return nil, err
	}

	primaryClient, err := dialFuncs[config.BlockchainType](config.ProviderUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := utils.GetChainID(ctx, primaryClient)
	if err != nil {
		log.Error().Err(err).Msg("failed to get chain id based on:" + config.ProviderUrl)
		return nil, err
	}
	clients := make([]utils.ClientInterface, 0)
	clients = append(clients, primaryClient)

	if config.UseAdditionalProviderUrls {
		providerUrls, providerUrlLoadErr := utils.LoadProviderUrls(ctx, int(chainID.Int64()))
		if providerUrlLoadErr != nil {
			log.Warn().Err(providerUrlLoadErr).Msg("failed to load additional provider urls")
		}

		for _, url := range providerUrls {
			subClient, subClientErr := dialFuncs[config.BlockchainType](url)
			if subClientErr != nil {
				log.Error().Err(subClientErr).Msg("failed to dial sub client")
				continue
			}
			clients = append(clients, subClient)
		}
	}

	wallet := strings.TrimPrefix(config.ReporterPk, "0x")
	nonce, err := utils.GetNonceFromPk(ctx, wallet, primaryClient)
	if err != nil {
		return nil, err
	}
	noncemanager.Set(wallet, nonce)

	delegatorUrl := os.Getenv(EnvDelegatorUrl)
	if delegatorUrl == "" {
		log.Warn().Msg("delegator url not set")
	}

	return &ChainHelper{
		clients:      clients,
		wallet:       wallet,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
	}, nil
}

func (t *ChainHelper) Close() {
	for _, helperClient := range t.clients {
		helperClient.Close()
	}
}

func (t *ChainHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	if t.delegatorUrl == "" {
		return nil, errorSentinel.ErrChainDelegatorUrlNotFound
	}

	payload, err := utils.MakePayload(tx)
	if err != nil {
		return nil, err
	}

	result, err := request.Request[signedTx](
		request.WithEndpoint(t.delegatorUrl+DelegatorEndpoint),
		request.WithMethod("POST"),
		request.WithBody(payload),
		request.WithTimeout(DelegatorTimeout))
	if err != nil {
		log.Error().Err(err).Msg("failed to request sign from delegator")
		return nil, err
	}

	if result.SignedRawTx == nil {
		return nil, errorSentinel.ErrChainEmptySignedRawTx
	}
	return utils.HashToTx(*result.SignedRawTx)
}

func (t *ChainHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, nonce uint64, args ...interface{}) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeDirectTx(ctx, c, contractAddressHex, t.wallet, functionString, t.chainID, nonce, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, nonce uint64, args ...interface{}) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeFeeDelegatedTx(ctx, c, contractAddressHex, t.wallet, functionString, t.chainID, nonce, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

// SignTxByFeePayer: used for testing purpose
func (t *ChainHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return utils.SignTxByFeePayer(ctx, tx, t.chainID)
}

func (t *ChainHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	var result interface{}
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.ReadContract(ctx, c, functionString, contractAddressHex, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) ChainID() *big.Int {
	return t.chainID
}

func (t *ChainHelper) NumClients() int {
	return len(t.clients)
}

func (t *ChainHelper) PublicAddress() (common.Address, error) {
	// should get the public address of next reporter yet not move the index
	result := common.Address{}

	reporterPrivateKey := t.wallet
	privateKey, err := crypto.HexToECDSA(reporterPrivateKey)
	if err != nil {
		return result, err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return result, errorSentinel.ErrChainPubKeyToECDSAFail
	}
	result = crypto.PubkeyToAddress(*publicKeyECDSA)
	return result, nil
}

func (t *ChainHelper) PublicAddressString() (string, error) {
	address, err := t.PublicAddress()
	if err != nil {
		return "", err
	}

	return address.Hex(), nil
}

func (t *ChainHelper) SubmitDelegatedFallbackDirect(ctx context.Context, contractAddress string, functionString string, args ...interface{}) error {
	var err error
	var tx *types.Transaction

	maxRetrial := 3
	clientIndex := 0

	nonce, err := noncemanager.GetAndIncrementNonce(t.wallet)
	if err != nil {
		return err
	}

	if t.delegatorUrl != "" {
		for i := 0; i < maxRetrial; i++ {
			tx, err = utils.MakeFeeDelegatedTx(ctx, t.clients[clientIndex], contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
			if err != nil {
				if utils.ShouldRetryWithSwitchedJsonRPC(err) {
					clientIndex = (clientIndex + 1) % len(t.clients)
				}
				continue
			}

			tx, err = t.GetSignedFromDelegator(tx)
			if err != nil {
				continue
			}

			err = utils.SubmitRawTx(ctx, t.clients[clientIndex], tx)
			if err != nil {
				if utils.ShouldRetryWithSwitchedJsonRPC(err) {
					clientIndex = (clientIndex + 1) % len(t.clients)
				} else if errors.Is(err, errorSentinel.ErrChainTransactionFail) || utils.IsNonceError(err) {
					nonce, err = noncemanager.GetAndIncrementNonce(t.wallet)
					if err != nil {
						return err
					}
				}
				continue
			}
			return nil
		}
	}

	for i := 0; i < maxRetrial; i++ {
		tx, err = utils.MakeDirectTx(ctx, t.clients[clientIndex], contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
		if err != nil {
			if utils.ShouldRetryWithSwitchedJsonRPC(err) {
				clientIndex = (clientIndex + 1) % len(t.clients)
			}
			continue
		}

		err = utils.SubmitRawTx(ctx, t.clients[clientIndex], tx)
		if err != nil {
			if utils.ShouldRetryWithSwitchedJsonRPC(err) {
				clientIndex = (clientIndex + 1) % len(t.clients)
			} else if errors.Is(err, errorSentinel.ErrChainTransactionFail) || utils.IsNonceError(err) {
				nonce, err = noncemanager.GetAndIncrementNonce(t.wallet)
				if err != nil {
					return err
				}
			}
			continue
		}
		return nil
	}

	return err
}

func (t *ChainHelper) retryOnJsonRpcFailure(ctx context.Context, job func(c utils.ClientInterface) error) error {
	for _, client := range t.clients {
		err := job(client)
		if err != nil {
			if utils.ShouldRetryWithSwitchedJsonRPC(err) {
				log.Error().Err(err).Msg("Error on retrying on JsonRpcFailure")
				continue
			}
			return err
		}
		break
	}
	return nil
}
