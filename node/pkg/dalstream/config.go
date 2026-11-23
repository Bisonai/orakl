// Package dalstream pushes DAL prices to browsers over Server-Sent Events.
// It keeps one WebSocket to DAL, collapses its per-symbol messages and, once
// per tick, sends each client only the requested symbols whose value changed.
package dalstream

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env names.
const (
	EnvDALURL           = "DAL_URL"
	EnvDALAPIKey        = "DAL_API_KEY"
	EnvAddr             = "DAL_STREAM_ADDR"
	EnvMetricsAddr      = "DAL_STREAM_METRICS_ADDR"
	EnvTick             = "DAL_STREAM_TICK"
	EnvMaxConns         = "DAL_STREAM_MAX_CONNECTIONS"
	EnvMaxConnsPerIP    = "DAL_STREAM_MAX_CONNECTIONS_PER_IP"
	EnvTrustedProxies   = "DAL_STREAM_TRUSTED_PROXIES"
	EnvAllowedOrigins   = "DAL_STREAM_ALLOWED_ORIGINS"
	EnvSymbolsRefresh   = "DAL_STREAM_SYMBOLS_REFRESH"
	EnvUpstreamStaleAge = "DAL_STREAM_STALE_AFTER"
)

// Fixed limits, not configurable by env.
const (
	DefaultPingInterval = 15 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultClientQueue  = 8
	// RetryMillis is the SSE reconnect hint sent to browsers.
	RetryMillis = 3000
)

type Config struct {
	DALURL string // http(s) base of DAL, e.g. http://dal:8090
	APIKey string

	Addr        string
	MetricsAddr string

	Tick           time.Duration
	MaxConns       int
	MaxConnsPerIP  int
	TrustedProxies []netip.Prefix
	AllowedOrigins map[string]struct{}

	SymbolsRefresh time.Duration
	// StaleAfter: no upstream message for this long makes /health fail and
	// the upstream connection reconnect.
	StaleAfter time.Duration

	PingInterval time.Duration
	WriteTimeout time.Duration
	ClientQueue  int
	MinBackoff   time.Duration
	MaxBackoff   time.Duration
}

// DefaultConfig returns the defaults; DALURL and APIKey stay empty.
func DefaultConfig() Config {
	return Config{
		Addr:           ":8080",
		MetricsAddr:    ":8081",
		Tick:           time.Second,
		MaxConns:       1000,
		MaxConnsPerIP:  10,
		AllowedOrigins: map[string]struct{}{},
		SymbolsRefresh: 5 * time.Minute,
		StaleAfter:     10 * time.Second,
		PingInterval:   DefaultPingInterval,
		WriteTimeout:   DefaultWriteTimeout,
		ClientQueue:    DefaultClientQueue,
		MinBackoff:     500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
	}
}

// ConfigFromEnv reads the config from the environment.
func ConfigFromEnv() (Config, error) {
	cfg := DefaultConfig()
	cfg.DALURL = strings.TrimRight(strings.TrimSpace(os.Getenv(EnvDALURL)), "/")
	cfg.APIKey = strings.TrimSpace(os.Getenv(EnvDALAPIKey))
	if v := os.Getenv(EnvAddr); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv(EnvMetricsAddr); v != "" {
		cfg.MetricsAddr = v
	}

	var errs []error
	durEnv := func(name string, dst *time.Duration) {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			return
		}
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			errs = append(errs, fmt.Errorf("%s: want a positive duration, got %q", name, v))
			return
		}
		*dst = d
	}
	intEnv := func(name string, dst *int) {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			return
		}
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			errs = append(errs, fmt.Errorf("%s: want a positive integer, got %q", name, v))
			return
		}
		*dst = n
	}
	durEnv(EnvTick, &cfg.Tick)
	durEnv(EnvSymbolsRefresh, &cfg.SymbolsRefresh)
	durEnv(EnvUpstreamStaleAge, &cfg.StaleAfter)
	intEnv(EnvMaxConns, &cfg.MaxConns)
	intEnv(EnvMaxConnsPerIP, &cfg.MaxConnsPerIP)

	proxies, err := ParseTrustedProxies(os.Getenv(EnvTrustedProxies))
	if err != nil {
		errs = append(errs, err)
	}
	cfg.TrustedProxies = proxies
	cfg.AllowedOrigins = ParseOrigins(os.Getenv(EnvAllowedOrigins))

	if err := cfg.Validate(); err != nil {
		errs = append(errs, err)
	}
	return cfg, errors.Join(errs...)
}

// Validate checks the required fields.
func (c Config) Validate() error {
	var errs []error
	if c.APIKey == "" {
		errs = append(errs, fmt.Errorf("%s is required", EnvDALAPIKey))
	}
	if c.DALURL == "" {
		errs = append(errs, fmt.Errorf("%s is required", EnvDALURL))
	} else if u, err := url.Parse(c.DALURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, fmt.Errorf("%s: want an http(s) URL, got %q", EnvDALURL, c.DALURL))
	}
	return errors.Join(errs...)
}

// ParseTrustedProxies parses a comma-separated list of IPs and CIDRs.
func ParseTrustedProxies(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", EnvTrustedProxies, err)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", EnvTrustedProxies, err)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// ParseOrigins parses a comma-separated list of exact origins
// (scheme://host[:port], no trailing slash).
func ParseOrigins(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimRight(strings.TrimSpace(part), "/")
		if part != "" {
			out[part] = struct{}{}
		}
	}
	return out
}

// wsURL maps the DAL http(s) base to its /ws endpoint.
func wsURL(base string) string {
	switch {
	case strings.HasPrefix(base, "https://"):
		return "wss://" + strings.TrimPrefix(base, "https://") + "/ws"
	case strings.HasPrefix(base, "http://"):
		return "ws://" + strings.TrimPrefix(base, "http://") + "/ws"
	}
	return base + "/ws"
}

func isTrusted(addr netip.Addr, proxies []netip.Prefix) bool {
	for _, p := range proxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// clientIP returns the client address. X-Forwarded-For is used only when the
// direct peer is a trusted proxy; it is walked right to left and the first
// address that is not a trusted proxy wins.
func clientIP(remoteAddr string, xff []string, proxies []netip.Prefix) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !isTrusted(peer, proxies) {
		return peer.String()
	}
	var hops []string
	for _, h := range xff {
		for _, part := range strings.Split(h, ",") {
			if part = strings.TrimSpace(part); part != "" {
				hops = append(hops, part)
			}
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(hops[i])
		if err != nil {
			// garbage in the chain: stop at the last address we could trust
			break
		}
		a = a.Unmap()
		if !isTrusted(a, proxies) {
			return a.String()
		}
		peer = a
	}
	return peer.String()
}
