package metrics

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	adminmetrics "bisonai.com/miko/node/pkg/admin/metrics"
	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

const (
	// EnvAddr is the metrics listen address; empty disables metrics.
	EnvAddr = "METRICS_ADDR"
	// PollInterval is how often the fee payer balance and allowlist size are
	// refreshed.
	PollInterval = 60 * time.Second
	pollTimeout  = 10 * time.Second
)

// Start serves GET /metrics (default registry, incl. Go and process
// collectors) on addr in the background. It never mounts API routes. The
// address is bound synchronously so a bind failure is returned to the caller.
// An empty addr disables the listener and returns (nil, nil).
func Start(addr string) (*http.Server, error) {
	srv, _, err := start(addr)
	return srv, err
}

func start(addr string) (*http.Server, net.Addr, error) {
	if addr == "" {
		return nil, nil, nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bind %s %s: %w", EnvAddr, addr, err)
	}
	log.Info().Str("addr", ln.Addr().String()).Msg("delegator metrics server started")
	return adminmetrics.Serve(ln), ln.Addr(), nil
}

// BalanceReader is the part of the RPC client the balance poller needs.
type BalanceReader interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

// StartBalancePoller polls the fee payer balance every interval in the
// background until ctx is done. address returns the current fee payer (false
// while no key is loaded). An empty rpcURL disables the poller. RPC errors are
// counted and retried on the next tick.
func StartBalancePoller(ctx context.Context, rpcURL string, address func() (common.Address, bool), interval time.Duration) {
	if rpcURL == "" {
		log.Info().Msg("PROVIDER_URL not set, fee payer balance metric disabled")
		return
	}
	var c *client.Client
	go poll(ctx, interval, func(ctx context.Context) {
		if c == nil {
			var err error
			c, err = client.DialContext(ctx, rpcURL)
			if err != nil {
				rpcError(RPCMethodDial)
				log.Warn().Err(err).Msg("failed to dial RPC for fee payer balance")
				c = nil
				return
			}
		}
		if !pollBalance(ctx, c, address) {
			// redialed next tick: a dropped ws connection never recovers
			c.Close()
			c = nil
		}
	})
}

// StartAllowlistPoller refreshes the allowlist size every interval in the
// background until ctx is done.
func StartAllowlistPoller(ctx context.Context, count func(context.Context) (int, error), interval time.Duration) {
	go poll(ctx, interval, func(ctx context.Context) {
		n, err := count(ctx)
		if err != nil {
			StoreError(StoreOpAllowlistCount)
			log.Warn().Err(err).Msg("failed to count allowlist entries")
			return
		}
		SetAllowlistEntries(n)
	})
}

// poll runs fn at once and then every interval until ctx is done.
func poll(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		pctx, cancel := context.WithTimeout(ctx, pollTimeout)
		fn(pctx)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// pollBalance reports false on an RPC error.
func pollBalance(ctx context.Context, r BalanceReader, address func() (common.Address, bool)) bool {
	addr, ok := address()
	if !ok {
		return true
	}
	peb, err := r.BalanceAt(ctx, addr, nil)
	if err != nil {
		rpcError(RPCMethodBalanceAt)
		log.Warn().Err(err).Msg("failed to read fee payer balance")
		return false
	}
	setFeePayerBalance(PebToKaia(peb))
	return true
}

// PebToKaia converts peb (1e-18 KAIA) to KAIA.
func PebToKaia(peb *big.Int) float64 {
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(peb), big.NewFloat(1e18)).Float64()
	return f
}
