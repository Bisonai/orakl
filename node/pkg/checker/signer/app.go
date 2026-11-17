package signer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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

// Seams for unit tests; production code uses the real chain reads.
var (
	getSignerAddresses            = GetSignerAddresses
	extractExpirationFromContract = ExtractExpirationFromContract
)

func setUp(ctx context.Context) error {
	var err error
	signerCheckInterval, err = time.ParseDuration(os.Getenv("SIGNER_CHECK_INTERVAL"))
	if err == nil && signerCheckInterval <= 0 {
		// time.NewTicker panics on non-positive intervals.
		err = fmt.Errorf("non-positive interval %s", signerCheckInterval)
	}
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

	// check never fails: read errors are counted and retried on the next tick.
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

// check exports signer expirations; alerts are Prometheus rules on orakl_signer_*.
func check(ctx context.Context) {
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	signerAddresses, err := getSignerAddresses(readCtx, jsonRpcUrl, submissionProxyContractAddr)
	cancel()
	if err != nil {
		// Transient RPC errors must not exit sentinel: count and retry on the next tick.
		log.Error().Err(err).Msg("Failed to get signer addresses")
		errorsCounter.Inc()
		return
	}
	log.Debug().Msgf("Signer addresses: %v", signerAddresses)
	if len(signerAddresses) == 0 {
		log.Warn().Msg("No signer addresses returned")
	}

	// drop signers no longer registered
	expiryGauge.Reset()
	countGauge.Set(float64(len(signerAddresses)))

	failed := false
	for _, signerAddress := range signerAddresses {
		readCtx, cancel := context.WithTimeout(ctx, readTimeout)
		exp, err := extractExpirationFromContract(readCtx, jsonRpcUrl, submissionProxyContractAddr, signerAddress)
		cancel()
		if err == nil && exp == nil {
			err = errors.New("nil expiration returned")
		}
		if err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("Failed to extract expiration for signer: %s", signerAddress))
			errorsCounter.Inc()
			failed = true
			continue
		}
		log.Debug().Msgf("Signer: %s, Expiration: %s", signerAddress, exp.String())
		expiryGauge.WithLabelValues(signerAddress).Set(float64(exp.Unix()))
	}
	if !failed {
		recordLastSuccess(time.Now())
	}
}
