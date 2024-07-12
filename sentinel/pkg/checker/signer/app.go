package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
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

func setUp(ctx context.Context) error {
	var err error
	signerCheckInterval, err = time.ParseDuration(os.Getenv("SIGNER_CHECK_INTERVAL"))
	if err != nil {
		signerCheckInterval = DEFAULT_SIGNER_CHECK_INTERVAL_HOUR * time.Hour
		log.Error().Err(err).Msgf("Using default signer check interval of %d hours", DEFAULT_SIGNER_CHECK_INTERVAL_HOUR)
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
	var err error
	err = setUp(ctx)
	if err != nil {
		return err
	}

	log.Info().Msg("Starting signer expiration checker")
	checkTicker := time.NewTicker(signerCheckInterval)
	defer checkTicker.Stop()

	err = check(ctx)
	if err != nil {
		return err
	}

	for range checkTicker.C {
		err = check(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func check(ctx context.Context) error {
	alarmMessage := ""
	signerAddresses, err := GetSignerAddresses(ctx, jsonRpcUrl, submissionProxyContractAddr)
	log.Info().Msgf("Signer addresses: %v", signerAddresses)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get signer addresses")
		return err
	}

	for _, signerAddress := range signerAddresses {
		exp, err := ExtractExpirationFromContract(ctx, jsonRpcUrl, submissionProxyContractAddr, signerAddress)
		log.Info().Msgf("Signer: %s, Expiration: %s", signerAddress, (*exp).String())
		if err != nil || exp == nil {
			log.Error().Err(err).Msg(fmt.Sprintf("Failed to extract expiration for signer: %s", signerAddress))
			alarmMessage += fmt.Sprintf("Failed to extract expiration for signer %s with following error: %v\n", signerAddress, err)
			continue
		}

		expiryDate := *exp
		now := time.Now()
		if now.Before(expiryDate) && now.Add(6*24*time.Hour).After(expiryDate) {
			alarmMessage += fmt.Sprintf("Auto signer renewal didn't work as expected for wallet %s. Expiry date: %s\n", signerAddress, exp.String())
		}
	}
	if alarmMessage != "" {
		alert.SlackAlert(alarmMessage)
	}

	return nil
}
