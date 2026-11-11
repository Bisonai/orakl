// Package configfetch fetches orakl-config files (e.g. mainnet_feeds.json,
// mag7/mainnet/AAPL.json) with retries, a fallback source and an optional
// last-known-good disk cache.
//
// Environment:
//   - CONFIG_BASE_URL: primary source (default DefaultBaseURL).
//   - CONFIG_FALLBACK_URL: secondary source, tried after the primary in every
//     round (default DefaultFallbackURL; set to an empty string to disable).
//   - CONFIG_CACHE_DIR: if set, every successful fetch is stored there and
//     served when all sources fail (e.g. so a restart survives an outage).
package configfetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	errorsentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/request"

	"github.com/rs/zerolog/log"
)

const (
	DefaultBaseURL     = "https://config.orakl.network"
	DefaultFallbackURL = "https://raw.githubusercontent.com/Bisonai/orakl-config/master"
)

// Retry policy (worst case ~63s: 3 rounds x 2 sources x 10s + 1s + 2s backoff,
// plus up to 25% jitter). Package vars so tests can shrink them.
var (
	rounds  = 3
	backoff = 1 * time.Second // doubled after each failed round
	timeout = 10 * time.Second
)

// FeedsPath is the path of the chain's feeds bundle.
func FeedsPath(chain string) string {
	return "/" + chain + "_feeds.json"
}

// URL is the primary-source URL of path (for logs).
func URL(path string) string {
	return baseURLs()[0] + path
}

// Get fetches path and decodes it as JSON into T.
func Get[T any](ctx context.Context, path string) (T, error) {
	var out T
	body, err := Fetch(ctx, path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("decode config %s: %w", path, err)
	}
	return out, nil
}

// Fetch returns the raw body of path (which must start with "/"). Each round
// tries every source in order; rounds repeat with backoff while any failure was
// transient (network error, timeout, HTTP 5xx/408/429). A body that is not
// valid JSON counts as a failure of that source. If all rounds fail and
// CONFIG_CACHE_DIR holds a previous copy, that copy is returned.
func Fetch(ctx context.Context, path string) ([]byte, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid config path %q", path)
	}
	bases := baseURLs()
	for _, b := range bases {
		if u, err := url.Parse(b); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, fmt.Errorf("invalid config base url %q", b)
		}
	}
	cacheDir := os.Getenv("CONFIG_CACHE_DIR")

	wait := backoff
	var errs []error
	round := 1
	for ; ; round++ {
		errs = errs[:0]
		retryable := false
		for _, base := range bases {
			endpoint := base + path
			body, r, err := fetchOnce(ctx, endpoint)
			if err == nil {
				if cacheDir != "" {
					writeCache(cacheDir, path, body)
				}
				return body, nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			log.Warn().Err(err).Int("round", round).Int("rounds", rounds).Str("url", endpoint).Msg("Failed to fetch config")
			errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
			retryable = retryable || r
		}
		if !retryable || round >= rounds {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(withJitter(wait)):
		}
		wait *= 2
	}

	fetchErr := fmt.Errorf("fetch config %s failed (round %d/%d): %w", path, round, rounds, errors.Join(errs...))
	if cacheDir != "" {
		if body, modTime, err := readCache(cacheDir, path); err == nil {
			log.Warn().Err(fetchErr).Str("path", path).Time("cachedAt", modTime).Msg("Using cached config")
			return body, nil
		}
	}
	return nil, fetchErr
}

func baseURLs() []string {
	primary := os.Getenv("CONFIG_BASE_URL")
	if primary == "" {
		primary = DefaultBaseURL
	}
	fallback, ok := os.LookupEnv("CONFIG_FALLBACK_URL")
	if !ok {
		fallback = DefaultFallbackURL
	}
	bases := []string{strings.TrimSuffix(primary, "/")}
	if fallback = strings.TrimSuffix(fallback, "/"); fallback != "" && fallback != bases[0] {
		bases = append(bases, fallback)
	}
	return bases
}

// fetchOnce performs a single GET. retryable reports whether the failure is
// transient.
func fetchOnce(ctx context.Context, endpoint string) (body []byte, retryable bool, err error) {
	resp, err := request.RequestRaw(
		request.WithContext(ctx),
		request.WithEndpoint(endpoint),
		request.WithTimeout(timeout))
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

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if !json.Valid(body) {
		const maxSnippet = 200
		snippet := body
		if len(snippet) > maxSnippet {
			snippet = snippet[:maxSnippet]
		}
		return nil, false, fmt.Errorf("invalid JSON (body: %q)", snippet)
	}
	return body, false, nil
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

func cacheFile(dir, path string) string {
	return filepath.Join(dir, strings.ReplaceAll(strings.TrimPrefix(path, "/"), "/", "_"))
}

// writeCache stores body atomically; failures are logged, never returned.
func writeCache(dir, path string, body []byte) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Warn().Err(err).Str("dir", dir).Msg("Failed to create config cache dir")
		return
	}
	tmp, err := os.CreateTemp(dir, ".configfetch-*")
	if err != nil {
		log.Warn().Err(err).Str("dir", dir).Msg("Failed to write config cache")
		return
	}
	_, err = tmp.Write(body)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), cacheFile(dir, path))
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		log.Warn().Err(err).Str("path", path).Msg("Failed to write config cache")
	}
}

func readCache(dir, path string) ([]byte, time.Time, error) {
	name := cacheFile(dir, path)
	info, err := os.Stat(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	body, err := os.ReadFile(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !json.Valid(body) {
		return nil, time.Time{}, fmt.Errorf("cached config %s is not valid JSON", name)
	}
	return body, info.ModTime(), nil
}
