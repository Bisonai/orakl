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

type SignHelperConfig struct {
	pk string
}

type SignHelperOption func(*SignHelperConfig)

func WithSignerPk(pk string) SignHelperOption {
	return func(config *SignHelperConfig) {
		config.pk = pk
	}
}

func getSignerPk(ctx context.Context, config SignHelperConfig) (string, error) {
	var pk string
	var err error
	if config.pk != "" {
		pk = strings.TrimPrefix(config.pk, "0x")
		return pk, nil
	} else {
		pk, err = utils.LoadSignerPk(ctx)
		if err != nil || pk == "" {
			log.Warn().Msg("failed to load signer from pgs")
		}

		if pk == "" {
			pk = secrets.GetSecret(SignerPk)
			if pk == "" {
				log.Error().Msg("signer pk not set")
				return "", errorSentinel.ErrChainSignerPKNotFound
			}
			err = utils.StoreSignerPk(ctx, pk)
			if err != nil {
				log.Warn().Msg("failed to store pk")
			}
		}

		return strings.TrimPrefix(pk, "0x"), nil
	}
}

func NewSignHelper(ctx context.Context, opts ...SignHelperOption) (*SignHelper, error) {
	config := SignHelperConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	pk, err := getSignerPk(ctx, config)
	if err != nil {
		log.Error().Err(err).Msg("failed to get signer pk")
		return nil, err
	}

	privateKey, err := utils.StringToPk(pk)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert pk")
		return nil, err
	}

	chainHelper, err := NewChainHelper(ctx, WithReporterPk(pk), WithoutAdditionalWallets())
	if err != nil {
		log.Error().Err(err).Msg("failed to set chainHelper for signHelper")
		return nil, err
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		log.Error().Msg("SUBMISSION_PROXY_CONTRACT not found, signer initialization failed")
		return nil, errorSentinel.ErrChainSubmissionProxyContractNotFound
	}

	signHelper := &SignHelper{
		PK: privateKey,

		chainHelper:                 chainHelper,
		submissionProxyContractAddr: submissionProxyContractAddr,
	}

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
	if s.expirationDate == nil || s.expirationDate.IsZero() {
		_, err := s.LoadExpiration(ctx)
		if err != nil {
			return err
		}
	}

	if !s.IsRenewalRequired() {
		return nil
	}

	newPK, newPkHex, err := utils.NewPk(ctx)
	if err != nil {
		return err
	}

	return s.Renew(ctx, newPK, newPkHex)
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
		return nil, errorSentinel.ErrChainFailedToParseContractResult
	}
	rawTimestamp := values[1].(*big.Int)
	expirationDate := time.Unix(int64(rawTimestamp.Int64()), 0)
	s.expirationDate = &expirationDate
	return s.expirationDate, nil
}

func (s *SignHelper) IsRenewalRequired() bool {
	return time.Until(*s.expirationDate) < SignerRenewThreshold
}

func (s *SignHelper) Renew(ctx context.Context, newPK *ecdsa.PrivateKey, newPkHex string) error {
	newPublicAddr, err := utils.StringPkToAddressHex(newPkHex)
	if err != nil {
		return err
	}
	addr := common.HexToAddress(newPublicAddr)

	err = s.signerUpdate(ctx, addr)
	if err != nil {
		return err
	}
	log.Debug().Str("Player", "Signer").Msg("signer renewed from the contract")

	s.mu.Lock()
	s.PK = newPK
	s.mu.Unlock()

	s.chainHelper.Close()
	newChainHelper, err := NewChainHelper(ctx, WithReporterPk(newPkHex), WithoutAdditionalWallets())
	if err != nil {
		return err
	}
	s.chainHelper = newChainHelper

	_, err = s.LoadExpiration(ctx)
	if err != nil {
		return err
	}
	return utils.StoreSignerPk(ctx, newPkHex)
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
