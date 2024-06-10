package signer

import (
	"context"
	"errors"
	"os"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
	"github.com/rs/zerolog/log"
)

type RegsiteredSigner struct {
	Exp time.Time
}

var signerCheckInterval time.Duration
var signer RegsiteredSigner

func setUp(ctx context.Context) error {
	signerCheckInterval = 12 * time.Hour
	checkInterval := os.Getenv("SIGNER_CHECK_INTERVAL")
	parsedCheckInterval, err := time.ParseDuration(checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse SIGNER_CHECK_INTERVAL, using default 2h")
	} else {
		signerCheckInterval = parsedCheckInterval
	}

	nodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if nodeAdminUrl == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	signerAddr, err := request.GetRequest[string](nodeAdminUrl+"/wallet/signer", nil, nil)
	if err != nil {
		return err
	}

	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		return errors.New("JSON_RPC_URL not found")
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		return errors.New("SUBMISSION_PROXY_CONTRACT not found")
	}

	exp, err := ExtractExpirationFromContract(ctx, jsonRpcUrl, submissionProxyContractAddr, signerAddr)
	if err != nil {
		return err
	}

	signer = RegsiteredSigner{
		Exp: *exp,
	}
	return nil
}

func Start(ctx context.Context) error {
	err := setUp(ctx)
	if err != nil {
		return err
	}

	log.Info().Msg("Starting signer expiration checker")
	checkTicker := time.NewTicker(signerCheckInterval)
	defer checkTicker.Stop()

	for range checkTicker.C {
		check(ctx)
	}
	return nil
}

func check(ctx context.Context) {
	if time.Until(signer.Exp) < 7*24*time.Hour {
		remainingTime := time.Until(signer.Exp)
		alert.SlackAlert("Signer expires in: " + remainingTime.String())
	}
}
