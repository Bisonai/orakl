package helper

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"unicode"

	"bisonai.com/orakl/node/pkg/chain/noncemanagerv2"
	"bisonai.com/orakl/node/pkg/chain/utils"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/metrics/submission"
	"bisonai.com/orakl/node/pkg/secrets"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

func setProviderAndReporter(config *ChainHelperConfig, blockchainType BlockchainType) error {
	switch blockchainType {
	case Kaia:
		if config.ProviderUrl == "" {
			config.ProviderUrl = secrets.GetSecret(KaiaProviderUrl)
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
			config.ProviderUrl = secrets.GetSecret(EthProviderUrl)
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

	// the token is only used with a delegator: without DELEGATOR_URL it is
	// ignored; with it, a bad token fails every helper (fail fast at startup)
	delegatorUrl := os.Getenv(EnvDelegatorUrl)
	var delegatorToken string
	if delegatorUrl != "" {
		delegatorToken, err = parseDelegatorToken(secrets.GetSecret(DelegatorToken))
		if err != nil {
			return nil, err
		}
	}

	primaryClient, err := dialFuncs[config.BlockchainType](config.ProviderUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := utils.GetChainID(ctx, primaryClient)
	if err != nil {
		log.Error().Err(err).Msg("failed to get chain id based on:" + config.ProviderUrl)
		primaryClient.Close()
		return nil, err
	}

	wallet := strings.TrimPrefix(config.ReporterPk, "0x")

	nonceManager, err := noncemanagerv2.New(ctx, primaryClient, wallet)
	if err != nil {
		primaryClient.Close()
		return nil, err
	}

	return &ChainHelper{
		client:         primaryClient,
		wallet:         wallet,
		chainID:        chainID,
		delegatorUrl:   delegatorUrl,
		delegatorToken: delegatorToken,
		noncemanager:   nonceManager,
	}, nil
}

// parseDelegatorToken trims surrounding whitespace (e.g. a trailing newline
// from a secret file), which net/http would reject as a header value. A token
// still holding control characters fails here, naming the variable but never
// the value. Empty stays allowed: delegators without auth.
func parseDelegatorToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if strings.ContainsFunc(token, unicode.IsControl) {
		return "", fmt.Errorf("%s contains control characters", DelegatorToken)
	}
	return token, nil
}

func (t *ChainHelper) Close() {
	t.client.Close()
}

func (t *ChainHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	signed, _, err := t.signFromDelegator(tx)
	return signed, err
}

// signFromDelegator is GetSignedFromDelegator plus a bounded failure reason
// (submission.Reason*) for metrics; reason is empty on success or when no
// delegator is configured.
func (t *ChainHelper) signFromDelegator(tx *types.Transaction) (*types.Transaction, string, error) {
	if t.delegatorUrl == "" {
		// not a delegator failure: callers send direct (PathDirect)
		return nil, "", errorSentinel.ErrChainDelegatorUrlNotFound
	}

	payload, err := utils.MakePayload(tx)
	if err != nil {
		return nil, submission.ReasonRequest, err
	}

	opts := []request.RequestOption{
		request.WithEndpoint(t.delegatorUrl + DelegatorEndpoint),
		request.WithMethod("POST"),
		request.WithBody(payload),
		request.WithTimeout(DelegatorTimeout),
	}
	if t.delegatorToken != "" {
		opts = append(opts, request.WithHeaders(map[string]string{"Authorization": "Bearer " + t.delegatorToken}))
	}
	result, err := request.Request[signedTx](opts...)
	if err != nil {
		// debug only: fallback callers warn rate-limited
		log.Debug().Err(err).Msg("failed to request sign from delegator")
		return nil, delegatorErrorReason(err), err
	}

	if result.SignedRawTx == nil {
		return nil, submission.ReasonInvalidResponse, errorSentinel.ErrChainEmptySignedRawTx
	}
	signed, err := utils.HashToTx(*result.SignedRawTx)
	if err != nil {
		return nil, submission.ReasonInvalidResponse, err
	}
	return signed, "", nil
}

// delegatorErrorReason maps a request.Request error to a bounded reason.
func delegatorErrorReason(err error) string {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.Is(err, errorSentinel.ErrRequestStatusNotOk):
		return submission.ReasonHTTPStatus
	case errors.As(err, &syntaxErr), errors.As(err, &typeErr):
		return submission.ReasonInvalidResponse
	default:
		return submission.ReasonRequest
	}
}

func (t *ChainHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	lease := t.noncemanager.Acquire()
	tx, err := utils.MakeDirectTx(ctx, t.client, contractAddressHex, t.wallet, functionString, t.chainID, lease.Nonce, args...)
	if err != nil {
		// build failed, nothing sent: hand the nonce back to avoid a gap
		t.noncemanager.Release(lease)
		return nil, err
	}
	return tx, nil
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
	return t.submitDelegatedFallbackDirect(ctx, submission.PurposeSubmit, contractAddress, functionString, args...)
}

// submitDelegatedFallbackDirect sends a fee-delegated tx, or a direct tx paid
// by t's key when the delegator is unset or fails. Each attempt is counted by
// path and purpose once the path is known (before build/broadcast); a delegator failure is counted and warned (rate-limited).
func (t *ChainHelper) submitDelegatedFallbackDirect(ctx context.Context, purpose, contractAddress, functionString string, args ...interface{}) error {
	lease := t.noncemanager.Acquire()
	nonce := lease.Nonce
	log.Debug().Uint64("nonce", nonce).Msg("nonce")

	// Release the nonce only on errors before SubmitRawTx (nothing sent; the
	// delegator only signs). Send errors are ambiguous (the tx may be in the
	// txpool, or the node rejected the nonce as too low), so keep them as is.
	tx, err := utils.MakeFeeDelegatedTx(ctx, t.client, contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
	if err != nil {
		t.noncemanager.Release(lease)
		return err
	}

	signed, path := t.signOrFallback(tx, purpose)
	submission.RecordSubmission(path, purpose)
	if signed == nil {
		signed, err = utils.MakeDirectTx(ctx, t.client, contractAddress, t.wallet, functionString, t.chainID, nonce, args...)
		if err != nil {
			t.noncemanager.Release(lease)
			return err
		}
	}

	if t.beforeSend != nil {
		t.beforeSend()
	}
	return utils.SubmitRawTx(ctx, t.client, signed)
}

// signOrFallback asks the delegator to sign tx. It returns the signed tx and
// PathDelegated, or nil and the direct path to fall back on: PathDirect when no
// delegator is configured, else PathDirectFallback after counting and warning
// about the delegator error.
func (t *ChainHelper) signOrFallback(tx *types.Transaction, purpose string) (*types.Transaction, string) {
	if t.delegatorUrl == "" {
		return nil, submission.PathDirect
	}
	signed, reason, err := t.signFromDelegator(tx)
	if err == nil {
		return signed, submission.PathDelegated
	}
	submission.RecordDelegatorError(reason, purpose)
	fallbackWarner.warn(reason, purpose, err)
	return nil, submission.PathDirectFallback
}

func (t *ChainHelper) SubmitDirect(ctx context.Context, contractAddress, functionString string, args ...interface{}) error {
	tx, err := t.MakeDirectTx(ctx, contractAddress, functionString, args...)
	if err != nil {
		return err
	}

	return utils.SubmitRawTx(ctx, t.client, tx)
}

func (t *ChainHelper) FlushNoncePool(ctx context.Context) error {
	return t.noncemanager.Reset(ctx)
}
