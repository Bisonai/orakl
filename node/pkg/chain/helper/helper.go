package helper

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"
	"strings"

	"bisonai.com/miko/node/pkg/chain/noncemanagerv2"
	"bisonai.com/miko/node/pkg/chain/utils"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
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
		BlockchainType: Kaia,
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

	wallet := strings.TrimPrefix(config.ReporterPk, "0x")

	nonceManager := noncemanagerv2.New(primaryClient)
	go nonceManager.StartAutoRefill(ctx)

	delegatorUrl := os.Getenv(EnvDelegatorUrl)

	return &ChainHelper{
		client:       primaryClient,
		wallet:       wallet,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
		noncemanager: nonceManager,
	}, nil
}

func (t *ChainHelper) Close() {
	t.client.Close()
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

func (t *ChainHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	nonce, err := t.noncemanager.GetNonce(ctx, t.wallet)
	if err != nil {
		return nil, err
	}

	return utils.MakeDirectTx(ctx, t.client, contractAddressHex, t.wallet, functionString, t.chainID, nonce, args...)
}

func (t *ChainHelper) Submit(ctx context.Context, tx *types.Transaction) error {
	return utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *ChainHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, nonce uint64, args ...interface{}) (*types.Transaction, error) {
	return utils.MakeFeeDelegatedTx(ctx, t.client, contractAddressHex, t.wallet, functionString, t.chainID, nonce, args...)
}

// SignTxByFeePayer: used for testing purpose
func (t *ChainHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return utils.SignTxByFeePayer(ctx, tx, t.chainID)
}

func (t *ChainHelper) ReadContract(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (interface{}, error) {
	return utils.ReadContract(ctx, t.client, functionString, contractAddressHex, args...)
}

func (t *ChainHelper) ChainID() *big.Int {
	return t.chainID
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

func (t *ChainHelper) SubmitDelegatedFallbackDirect(ctx context.Context, contractAddress, functionString string, args ...interface{}) error {
	nonce, err := t.noncemanager.GetNonce(ctx, t.wallet)
	if err != nil {
		return err
	}
	log.Debug().Uint64("nonce", nonce).Msg("nonce")

	tx, err := utils.MakeFeeDelegatedTx(ctx, t.client, contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
	if err != nil {
		return err
	}

	tx, err = t.GetSignedFromDelegator(tx)
	if err != nil {
		tx, err = utils.MakeDirectTx(ctx, t.client, contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
		if err != nil {
			return err
		}

		return utils.SubmitRawTx(ctx, t.client, tx)
	}

	return utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *ChainHelper) SubmitDirect(ctx context.Context, contractAddress, functionString string, args ...interface{}) error {
	tx, err := t.MakeDirectTx(ctx, contractAddress, functionString, args...)
	if err != nil {
		return err
	}

	return utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *ChainHelper) FlushNoncePool(ctx context.Context) error {
	return t.noncemanager.Refill(ctx, t.wallet)
}
