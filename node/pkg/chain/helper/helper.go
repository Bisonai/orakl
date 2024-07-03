package helper

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"
	"strings"
	"time"

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
		UseAdditionalWallets:      true,
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

	wallets := make([]string, 0)
	if config.UseAdditionalWallets {
		wallets, err = utils.GetWallets(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to get additional wallets")
		}
	}
	if config.ReporterPk != "" {
		primaryWallet := strings.TrimPrefix(config.ReporterPk, "0x")
		exists := false
		for _, wallet := range wallets {
			if wallet == primaryWallet {
				exists = true
				break
			}
		}

		if !exists {
			wallets = append([]string{primaryWallet}, wallets...)
			err = utils.InsertWallet(ctx, primaryWallet)
			if err != nil {
				log.Warn().Err(err).Msg("failed to insert primary wallet")
			}
		}
	}

	delegatorUrl := os.Getenv(EnvDelegatorUrl)
	if delegatorUrl == "" {
		log.Warn().Msg("delegator url not set")
	}

	return &ChainHelper{
		clients:      clients,
		wallets:      wallets,
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

	result, err := request.Request[signedTx](request.WithEndpoint(t.delegatorUrl+DelegatorEndpoint), request.WithMethod("POST"), request.WithBody(payload))
	if err != nil {
		log.Error().Err(err).Msg("failed to request sign from delegator")
		return nil, err
	}

	if result.SignedRawTx == nil {
		return nil, errorSentinel.ErrChainEmptySignedRawTx
	}
	return utils.HashToTx(*result.SignedRawTx)
}

func (t *ChainHelper) NextReporter() string {
	if len(t.wallets) == 0 {
		return ""
	}
	reporter := t.wallets[t.lastUsedWalletIndex]
	t.lastUsedWalletIndex = (t.lastUsedWalletIndex + 1) % len(t.wallets)
	return reporter
}

func (t *ChainHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeDirectTx(ctx, c, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, gasMultiplier float64, args ...interface{}) (*types.Transaction, error) {
	var result *types.Transaction
	job := func(c utils.ClientInterface) error {
		tmp, err := utils.MakeFeeDelegatedTx(ctx, c, contractAddressHex, t.NextReporter(), functionString, t.chainID, gasMultiplier, args...)
		if err == nil {
			result = tmp
		}
		return err
	}
	err := t.retryOnJsonRpcFailure(ctx, job)
	return result, err
}

func (t *ChainHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	job := func(c utils.ClientInterface) error {
		return utils.SubmitRawTx(ctx, c, tx)
	}
	return t.retryOnJsonRpcFailure(ctx, job)
}

func (t *ChainHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	job := func(c utils.ClientInterface) error {
		return utils.SubmitRawTxString(ctx, c, rawTx)
	}
	return t.retryOnJsonRpcFailure(ctx, job)
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

	reporterPrivateKey := t.wallets[t.lastUsedWalletIndex]
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

func (t *ChainHelper) retryOnJsonRpcFailure(ctx context.Context, job func(c utils.ClientInterface) error) error {
	for _, client := range t.clients {
		err := job(client)
		if err != nil {
			if utils.ShouldRetryWithSwitchedJsonRPC(err) {
				continue
			}
			return err
		}
		break
	}
	return nil
}

func NewSignHelper(ctx context.Context) (*SignHelper, error) {
	pk, err := utils.LoadSignerPk(ctx)
	if err != nil || pk == "" {
		log.Warn().Msg("failed to load signer pk from pgs")
	}

	if pk == "" {
		pk = secrets.GetSecret(SignerPk)
		if pk == "" {
			log.Error().Msg("signer pk not set")
			return nil, errorSentinel.ErrChainSignerPKNotFound
		}
		err = utils.StoreSignerPk(ctx, pk)
		if err != nil {
			log.Warn().Msg("failed to store pk")
		}
	}

	pk = strings.TrimPrefix(pk, "0x")
	privateKey, err := utils.StringToPk(pk)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert pk")
		return nil, err
	}

	chainHelper, err := NewChainHelper(ctx, WithReporterPk(pk), WithoutAdditionalWallets())
	if err != nil {
		log.Error().Err(err).Msg("failed to set chainHelper for signerfeat")
		log.Error().Err(err).Msg("failed to set chainHelper for signHelper")
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		log.Error().Msg("SUBMISSION_PROXY_CONTRACT not found")
		return nil, errorSentinel.ErrChainSubmissionProxyContractNotFound
	}

	signHelper := &SignHelper{
		PK: privateKey,

		chainHelper:                 chainHelper,
		submissionProxyContractAddr: submissionProxyContractAddr,
	}

	expirationDate, err := signHelper.LoadExpiration(ctx)
	if err != nil {
		return nil, err
	}
	signHelper.expirationDate = expirationDate

	go signHelper.autoRenew(ctx)

	return signHelper, nil
}

func (s *SignHelper) MakeGlobalAggregateProof(val int64, timestamp time.Time, name string) ([]byte, error) {
	s.mu.RLock()
	pk := s.PK
	s.mu.RUnlock()
	return utils.MakeValueSignature(val, timestamp.Unix(), name, pk)
}

func (s *SignHelper) autoRenew(ctx context.Context) {
	autoRenewTicker := time.NewTicker(SignerRenewInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-autoRenewTicker.C:
			err := s.CheckAndUpdateSignerPK(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to renew signer pk")
			}
		}
	}
}

func (s *SignHelper) CheckAndUpdateSignerPK(ctx context.Context) error {
	if !s.IsRenewalRequired() {
		return nil
	}
	err := s.Renew(ctx)
	if err != nil {
		return err
	}

	newExpiration, err := s.LoadExpiration(ctx)
	if err != nil {
		return err
	}
	s.expirationDate = newExpiration
	return nil
}

func (s *SignHelper) LoadExpiration(ctx context.Context) (*time.Time, error) {
	publicKey := s.PK.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errorSentinel.ErrChainPubKeyToECDSAFail
	}
	addr := crypto.PubkeyToAddress(*publicKeyECDSA)

	readResult, err := s.chainHelper.ReadContract(ctx, s.submissionProxyContractAddr, SignerDetailFuncSignature, addr)
	if err != nil {
		return nil, err
	}

	values, ok := readResult.([]interface{})
	if !ok {
		return nil, fmt.Errorf("interface conversion failure")
	}
	rawTimestamp := values[1].(*big.Int)
	expirationDate := time.Unix(int64(rawTimestamp.Int64()), 0)

	return &expirationDate, nil
}

func (s *SignHelper) IsRenewalRequired() bool {
	return time.Until(*s.expirationDate) < SignerRenewThreshold
}

func (s *SignHelper) Renew(ctx context.Context) error {
	newPK, newPkStr, err := utils.NewPk(ctx)
	if err != nil {
		return err
	}

	newPublicAddr, err := utils.StringPkToAddressHex(newPkStr)
	if err != nil {
		return err
	}
	addr := common.HexToAddress(newPublicAddr)

	err = s.signerUpdate(ctx, addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.PK = newPK
	s.mu.Unlock()

	return utils.StoreSignerPk(ctx, newPkStr)
}

func (s *SignHelper) signerUpdate(ctx context.Context, newAddr common.Address) error {
	if s.chainHelper.delegatorUrl != "" {
		return s.delegatedSignerUpdate(ctx, newAddr)
	}
	return s.directSignerUpdate(ctx, newAddr)
}

func (s *SignHelper) delegatedSignerUpdate(ctx context.Context, newAddr common.Address) error {
	rawTx, err := s.chainHelper.MakeFeeDelegatedTx(ctx, s.submissionProxyContractAddr, UpdateSignerFuncSignature, 0, newAddr)
	if err != nil {
		return err
	}
	signedTx, err := s.chainHelper.GetSignedFromDelegator(rawTx)
	if err != nil {
		return err
	}
	return s.chainHelper.SubmitRawTx(ctx, signedTx)
}

func (s *SignHelper) directSignerUpdate(ctx context.Context, newAddr common.Address) error {
	rawTx, err := s.chainHelper.MakeDirectTx(ctx, s.submissionProxyContractAddr, UpdateSignerFuncSignature, newAddr)
	if err != nil {
		return err
	}
	return s.chainHelper.SubmitRawTx(ctx, rawTx)
}
