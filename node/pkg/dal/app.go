package dal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/apiv2"
	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/request"

	"github.com/rs/zerolog/log"
)

type Config = types.Config

// baseMikoConfigUrl is the config bundle URL format (chain substituted).
// In-package tests override it to point at a local httptest server;
// production always uses the default.
var baseMikoConfigUrl = "https://config.orakl.network/%s_feeds.json"

// Startup config fetch retry policy (worst case ~40-44s: 5x5s + 1+2+4+8s
// backoff, plus up to 25% jitter).
// Package vars so tests can shrink them.
var (
	fetchConfigsAttempts = 5
	fetchConfigsBackoff  = 1 * time.Second // doubled after each failed attempt
	fetchConfigsTimeout  = 5 * time.Second // per attempt
)

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	keyStore, err := keystore.OpenFromEnv(ctx, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open key store")
		return err
	}
	defer keyStore.Close()

	if n, countErr := keyStore.Count(ctx); countErr != nil {
		log.Error().Err(countErr).Msg("Failed to count API keys")
	} else if n == 0 {
		log.Warn().Msg("Key store has no API keys; all authenticated requests will be rejected")
	} else {
		log.Info().Int("keys", n).Msg("Loaded key store")
	}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(ctx, 10*time.Minute)

	chain := os.Getenv("CHAIN")
	if chain == "" {
		log.Error().Msg("CHAIN environment variable not set")
		return errorsentinel.ErrDalChainEnvNotFound
	}

	configs, err := fetchConfigs(ctx, chain)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch configs")
		return err
	}

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}
	collector.Start(ctx)

	hub := hub.HubSetup(ctx, configs)
	go hub.Start(ctx, collector)

	err = apiv2.Start(ctx, apiv2.WithCollector(collector), apiv2.WithHub(hub), apiv2.WithKeyCache(keyCache), apiv2.WithKeyStore(keyStore))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start DAL WS server")
		return err
	}

	return nil
}

// fetchConfigs fetches the chain's config bundle, retrying transient
// failures (network errors, timeouts, HTTP 5xx/408/429) with exponential
// backoff. Other non-200 statuses, an invalid URL and JSON decode errors
// fail immediately. Cancellation returns ctx.Err().
func fetchConfigs(ctx context.Context, chain string) ([]Config, error) {
	endpoint := fmt.Sprintf(baseMikoConfigUrl, chain)
	if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid config url %q", endpoint)
	}
	backoff := fetchConfigsBackoff

	for attempt := 1; ; attempt++ {
		configs, retryable, err := fetchConfigsOnce(ctx, endpoint)
		if err == nil {
			return configs, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		if !retryable || attempt >= fetchConfigsAttempts {
			return nil, fmt.Errorf("fetch configs from %s failed (attempt %d/%d): %w", endpoint, attempt, fetchConfigsAttempts, err)
		}

		wait := withJitter(backoff)
		log.Warn().Err(err).
			Int("attempt", attempt).
			Int("attempts", fetchConfigsAttempts).
			Dur("backoff", wait).
			Str("url", endpoint).
			Msg("Failed to fetch configs, retrying")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		backoff *= 2
	}
}

// fetchConfigsOnce performs a single fetch. retryable reports whether the
// failure is transient.
func fetchConfigsOnce(ctx context.Context, endpoint string) (configs []Config, retryable bool, err error) {
	resp, err := request.RequestRaw(
		request.WithContext(ctx),
		request.WithEndpoint(endpoint),
		request.WithTimeout(fetchConfigsTimeout))
	if err != nil {
		return nil, true, err
	}
	defer func() {
		// Drain so the keep-alive connection can be reused by the next attempt.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, isRetryableStatus(resp.StatusCode), fmt.Errorf("%w: status %d", errorsentinel.ErrRequestStatusNotOk, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	if err := json.Unmarshal(body, &configs); err != nil {
		const maxSnippet = 200
		snippet := body
		if len(snippet) > maxSnippet {
			snippet = snippet[:maxSnippet]
		}
		return nil, false, fmt.Errorf("decode configs: %w (body: %q)", err, snippet)
	}
	return configs, false, nil
}

func isRetryableStatus(code int) bool {
	return code >= 500 || code == http.StatusRequestTimeout || code == http.StatusTooManyRequests
}

// withJitter adds up to 25% random jitter so replicas restarting together
// don't retry in lockstep.
func withJitter(d time.Duration) time.Duration {
	if d < 4 {
		return d
	}
	return d + rand.N(d/4)
}
