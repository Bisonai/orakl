package retrier

import (
	"crypto/rand"
	"math/big"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/rs/zerolog/log"
)

func Retry(job func() error, maxAttempts int, initialTimeout time.Duration, maxTimeout time.Duration) error {
	failureTimeout := initialTimeout
	for i := 0; i < maxAttempts; i++ {
		failureTimeout = calculateJitter(failureTimeout)
		if failureTimeout > maxTimeout {
			failureTimeout = maxTimeout
		}

		err := job()
		if err != nil {
			log.Error().Err(err).Msg("job failed, retrying")
			time.Sleep(failureTimeout)
			continue
		}
		return nil
	}
	log.Error().Msg("job failed")
	return errorSentinel.ErrRetrierJobFail
}

func calculateJitter(baseTimeout time.Duration) time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Error().Err(err).Msg("failed to generate jitter for retry timeout")
		return baseTimeout
	}
	jitter := time.Duration(n.Int64()) * time.Millisecond
	return baseTimeout + jitter
}
