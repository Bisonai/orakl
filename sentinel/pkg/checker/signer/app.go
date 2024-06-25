package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
	"github.com/rs/zerolog/log"
)

type RegisteredSigner struct {
	Name    string
	Address string
	Exp     time.Time
}

var signerCheckInterval time.Duration
var signers []RegisteredSigner

const ExpirationWarningThreshold = 7 * 24 * time.Hour

func setUp(ctx context.Context) error {
	var err error
	signerCheckInterval, err = time.ParseDuration(os.Getenv("SIGNER_CHECK_INTERVAL"))
	if err != nil {
		signerCheckInterval = 6 * time.Hour
		log.Error().Err(err).Msg("Using default signer check interval of 6 hours")
	}

	nodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if nodeAdminUrl == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	signerAddr, err := request.GetRequest[string](nodeAdminUrl+"/wallet/signer", nil, nil)
	if err != nil {
		return err
	}

	signers = append(signers,
		RegisteredSigner{
			Name:    "main",
			Address: signerAddr,
		},
	)

	if os.Getenv("SIGNER") != "" {
		addrs := strings.Split(os.Getenv("SIGNER"), ",")
		for _, addr := range addrs {
			signers = append(signers,
				RegisteredSigner{
					Name:    "sub",
					Address: addr,
				},
			)
		}
	}

	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		return errors.New("JSON_RPC_URL not found")
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		return errors.New("SUBMISSION_PROXY_CONTRACT not found")
	}

	for i, signer := range signers {
		exp, err := ExtractExpirationFromContract(ctx, jsonRpcUrl, submissionProxyContractAddr, signer.Address)
		if err != nil {
			continue
		}

		signers[i].Exp = *exp
	}
	return nil
}

func Start(ctx context.Context) error {
	err := setUp(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set up signer expiration checker")
		return err
	}

	log.Info().Msg("Starting signer expiration checker")
	checkTicker := time.NewTicker(signerCheckInterval)
	defer checkTicker.Stop()
	check(ctx)
	for range checkTicker.C {
		check(ctx)
	}
	return nil
}

func check(ctx context.Context) {
	for _, signer := range signers {
		log.Debug().Str("expiration", signer.Exp.String()).Msg("Checking signer expiration")
		if time.Until(signer.Exp) < ExpirationWarningThreshold {
			remainingTime := time.Until(signer.Exp)
			alert.SlackAlert(fmt.Sprintf("Signer %s(%s) expires in: %s", signer.Name, signer.Address, remainingTime.String()))
		}
	}
}
