package retrier

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
)

func Retry(job func() error, maxAttempts int, initialTimeout time.Duration, maxTimeout time.Duration) error {
	failureTimeout := initialTimeout
	var err error
	for i := 0; i < maxAttempts; i++ {
		failureTimeout = calculateJitter(failureTimeout)
		if failureTimeout > maxTimeout {
			failureTimeout = maxTimeout
		}

		err = job()
		if err != nil {
			time.Sleep(failureTimeout)
			continue
		}
		return nil
	}
	return err
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
