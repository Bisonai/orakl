package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/alert"
	"github.com/rs/zerolog/log"
)

type RegisteredSigner struct {
	Name    string
	Address string
	Exp     time.Time
}

var signerCheckInterval time.Duration
var jsonRpcUrl string
var submissionProxyContractAddr string

const DEFAULT_SIGNER_CHECK_INTERVAL_HOUR = 24

// readTimeout bounds each chain read so a hung RPC can't stall the checker forever.
const readTimeout = 30 * time.Second

// Seams for unit tests; production code uses the real chain reads and Slack alert.
var (
	getSignerAddresses            = GetSignerAddresses
	extractExpirationFromContract = ExtractExpirationFromContract
	slackAlert                    = alert.SlackAlert
)

func setUp(ctx context.Context) error {
	var err error
	signerCheckInterval, err = time.ParseDuration(os.Getenv("SIGNER_CHECK_INTERVAL"))
	if err != nil {
		signerCheckInterval = DEFAULT_SIGNER_CHECK_INTERVAL_HOUR * time.Hour
		log.Info().Err(err).Msgf("Using default signer check interval of %d hours", DEFAULT_SIGNER_CHECK_INTERVAL_HOUR)
	}

	jsonRpcUrl = os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		return errors.New("JSON_RPC_URL not found")
	}

	submissionProxyContractAddr = os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		return errors.New("SUBMISSION_PROXY_CONTRACT not found")
	}

	return nil
}

func Start(ctx context.Context) error {
	if err := setUp(ctx); err != nil {
		return err
	}

	log.Info().Msg("Starting signer expiration checker")
	checkTicker := time.NewTicker(signerCheckInterval)
	defer checkTicker.Stop()

	// check never fails: read errors are alerted and retried on the next tick.
	check(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-checkTicker.C:
			check(ctx)
		}
	}
}

func check(ctx context.Context) {
	alarmMessage := ""
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	signerAddresses, err := getSignerAddresses(readCtx, jsonRpcUrl, submissionProxyContractAddr)
	cancel()
	if err != nil {
		// Transient RPC errors must not exit sentinel: alert and retry on the next tick.
		log.Error().Err(err).Msg("Failed to get signer addresses")
		slackAlert(fmt.Sprintf("Signer checker failed to get signer addresses: %v", err))
		return
	}
	log.Debug().Msgf("Signer addresses: %v", signerAddresses)
	if len(signerAddresses) == 0 {
		log.Warn().Msg("No signer addresses returned")
		slackAlert("Signer checker found no signer addresses in submission proxy")
		return
	}

	for _, signerAddress := range signerAddresses {
		readCtx, cancel := context.WithTimeout(ctx, readTimeout)
		exp, err := extractExpirationFromContract(readCtx, jsonRpcUrl, submissionProxyContractAddr, signerAddress)
		cancel()
		if err == nil && exp == nil {
			err = errors.New("nil expiration returned")
		}
		if err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("Failed to extract expiration for signer: %s", signerAddress))
			alarmMessage += fmt.Sprintf("Failed to extract expiration for signer %s with following error: %v\n", signerAddress, err)
			continue
		}
		log.Debug().Msgf("Signer: %s, Expiration: %s", signerAddress, exp.String())

		expiryDate := *exp
		now := time.Now()
		if now.Before(expiryDate) && now.Add(6*24*time.Hour).After(expiryDate) {
			alarmMessage += fmt.Sprintf("Auto signer renewal didn't work as expected for wallet %s. Expiry date: %s\n", signerAddress, exp.String())
		}
	}
	if alarmMessage != "" {
		slackAlert(alarmMessage)
	}
}
