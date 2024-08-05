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
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/rs/zerolog/log"
)

type SignerConfig struct {
	pk             string
	renewInterval  time.Duration
	renewThreshold time.Duration
}

type SignerOption func(*SignerConfig)

func WithSignerPk(pk string) SignerOption {
	return func(config *SignerConfig) {
		config.pk = pk
	}
}

func WithRenewInterval(renewInterval time.Duration) SignerOption {
	return func(config *SignerConfig) {
		config.renewInterval = renewInterval
	}
}

func WithRenewThreshold(renewThreshold time.Duration) SignerOption {
	return func(config *SignerConfig) {
		config.renewThreshold = renewThreshold
	}
}

func getSignerPk(ctx context.Context, config SignerConfig) (string, error) {
	if config.pk != "" {
		return strings.TrimPrefix(config.pk, "0x"), nil
	}

	pk, err := utils.LoadSignerPk(ctx)
	if err != nil {
		log.Warn().Str("Player", "Signer").Err(err).Msg("failed to load signer from pgs")
	}

	if pk == "" {
		pk = secrets.GetSecret(SignerPk)
		if pk == "" {
			log.Error().Str("Player", "Signer").Msg("signer pk not set")
			return "", errorSentinel.ErrChainSignerPKNotFound
		}
		err = utils.StoreSignerPk(ctx, pk)
		if err != nil {
			log.Warn().Str("Player", "Signer").Err(err).Msg("failed to store pk")
		}
	}

	return strings.TrimPrefix(pk, "0x"), nil
}

func NewSigner(ctx context.Context, opts ...SignerOption) (*Signer, error) {
	config := SignerConfig{
		renewInterval:  DefaultSignerRenewInterval,
		renewThreshold: DefaultSignerRenewThreshold,
	}
	for _, opt := range opts {
		opt(&config)
	}

	pk, err := getSignerPk(ctx, config)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to get signer pk")
		return nil, err
	}

	privateKey, err := utils.StringToPk(pk)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to convert pk")
		return nil, err
	}

	chainHelper, err := NewChainHelper(
		ctx,
		WithReporterPk(pk))
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to set chainHelper for signHelper")
		return nil, err
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		log.Error().Str("Player", "Signer").Msg("SUBMISSION_PROXY_CONTRACT not found, signer initialization failed")
		return nil, errorSentinel.ErrChainSubmissionProxyContractNotFound
	}

	signHelper := &Signer{
		PK: privateKey,

		chainHelper:                 chainHelper,
		submissionProxyContractAddr: submissionProxyContractAddr,
		renewInterval:               config.renewInterval,
		renewThreshold:              config.renewThreshold,
	}

	go signHelper.autoRenew(ctx)

	return signHelper, nil
}

func (s *Signer) MakeGlobalAggregateProof(val int64, timestamp time.Time, name string) ([]byte, error) {
	s.mu.RLock()
	pk := s.PK
	s.mu.RUnlock()
	return utils.MakeValueSignature(val, timestamp.UnixMilli(), name, pk)
}

func (s *Signer) autoRenew(ctx context.Context) {
	autoRenewTicker := time.NewTicker(s.renewInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-autoRenewTicker.C:
			err := s.CheckAndUpdateSignerPK(ctx)
			if err != nil {
				log.Error().Str("Player", "Signer").Err(err).Msg("failed to renew signer pk")
			}
		}
	}
}

func (s *Signer) CheckAndUpdateSignerPK(ctx context.Context) error {
	if s.expirationDate == nil || s.expirationDate.IsZero() {
		_, err := s.LoadExpiration(ctx)
		if err != nil {
			log.Error().Str("Player", "Signer").Err(err).Msg("failed to load expiration date")
			return err
		}
	}

	if !s.IsRenewalRequired() {
		return nil
	}

	newPK, newPkHex, err := utils.NewPk(ctx)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to generate new pk")
		return err
	}

	return s.Renew(ctx, newPK, newPkHex)
}

func (s *Signer) LoadExpiration(ctx context.Context) (*time.Time, error) {
	publicKey := s.PK.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Error().Str("Player", "Signer").Err(errorSentinel.ErrChainPubKeyToECDSAFail).Msg("failed to convert pk")
		return nil, errorSentinel.ErrChainPubKeyToECDSAFail
	}
	addr := crypto.PubkeyToAddress(*publicKeyECDSA)

	readResult, err := s.chainHelper.ReadContract(ctx, s.submissionProxyContractAddr, SignerDetailFuncSignature, addr)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to read contract")
		return nil, err
	}

	values, ok := readResult.([]interface{})
	if !ok {
		log.Error().Str("Player", "Signer").Err(errorSentinel.ErrChainFailedToParseContractResult).Msg("failed to parse contract result")
		return nil, errorSentinel.ErrChainFailedToParseContractResult
	}

	rawTimestamp, ok := values[1].(*big.Int)
	if !ok {
		log.Error().Str("Player", "Signer").Err(errorSentinel.ErrChainFailedToParseContractResult).Msg("failed to parse result to bigInt")
		return nil, errorSentinel.ErrChainFailedToParseContractResult
	}

	expirationDate := time.Unix(int64(rawTimestamp.Int64()), 0)
	s.expirationDate = &expirationDate
	return s.expirationDate, nil
}

func (s *Signer) IsRenewalRequired() bool {
	return time.Until(*s.expirationDate) < s.renewThreshold
}

func (s *Signer) Renew(ctx context.Context, newPK *ecdsa.PrivateKey, newPkHex string) error {
	newPublicAddr, err := utils.StringPkToAddressHex(newPkHex)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to convert pk")
		return err
	}
	addr := common.HexToAddress(newPublicAddr)

	err = s.signerUpdate(ctx, addr)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to update signer")
		return err
	}
	log.Debug().Str("Player", "Signer").Msg("signer renewed from the contract")

	s.mu.Lock()
	s.PK = newPK
	s.mu.Unlock()

	s.chainHelper.Close()
	newChainHelper, err := NewChainHelper(
		ctx,
		WithReporterPk(newPkHex))
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to create new chain helper")
		return err
	}
	s.chainHelper = newChainHelper

	_, err = s.LoadExpiration(ctx)
	if err != nil {
		log.Error().Str("Player", "Signer").Err(err).Msg("failed to load expiration date")
		return err
	}
	return utils.StoreSignerPk(ctx, newPkHex)
}

func (s *Signer) signerUpdate(ctx context.Context, newAddr common.Address) error {
	return s.chainHelper.SubmitDelegatedFallbackDirect(ctx, s.submissionProxyContractAddr, UpdateSignerFuncSignature, newAddr)

}
