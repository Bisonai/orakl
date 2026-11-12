// Package onchain exports the on-chain state of every data feed as Prometheus
// metrics. It reads FeedUpdated events of all feeds with one eth_getLogs call
// per poll, so alerting and dashboards do not depend on an indexer.
package onchain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/utils/configfetch"
	"bisonai.com/miko/node/pkg/utils/configreload"
	kaia "github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/accounts/abi"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/client"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/common/math"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	IntervalEnv         = "ONCHAIN_CHECK_INTERVAL"
	DefaultPollInterval = 10 * time.Second
	// MaxBlockRange bounds one eth_getLogs call. After a longer outage the reader
	// skips ahead; every feed updates well within this window (~1000 s).
	MaxBlockRange = 1000
	// rpcConcurrency bounds parallel eth_calls when (re)loading feeds.
	rpcConcurrency = 8
)

var (
	feedUpdatedTopic = crypto.Keccak256Hash([]byte("FeedUpdated(int256)"))

	errMissingEnv = errors.New("missing env")
)

// Config is the subset of an orakl-config feed entry the reader needs.
type Config struct {
	Name           string `json:"name"`
	SubmitInterval *int   `json:"submitInterval"` // milliseconds
}

type chainClient interface {
	BlockNumber(ctx context.Context) (*big.Int, error)
	FilterLogs(ctx context.Context, q kaia.FilterQuery) ([]types.Log, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	CallContract(ctx context.Context, call kaia.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type feed struct {
	name     string
	address  common.Address
	interval time.Duration
}

type Reader struct {
	client chainClient
	proxy  common.Address

	feedsABI       *abi.ABI
	latestRoundABI *abi.ABI

	mu        sync.Mutex
	byAddress map[common.Address]feed
	addrCache map[string]common.Address
	lastBlock uint64
}

// Start runs the reader until ctx is done. Env: JSON_RPC_URL,
// SUBMISSION_PROXY_CONTRACT, CHAIN (default kairos), ONCHAIN_CHECK_INTERVAL.
func Start(ctx context.Context) error {
	rpcURL := os.Getenv("JSON_RPC_URL")
	proxy := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if rpcURL == "" || proxy == "" {
		return fmt.Errorf("%w: JSON_RPC_URL and SUBMISSION_PROXY_CONTRACT are required", errMissingEnv)
	}
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "kairos"
	}

	c, err := client.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer c.Close()

	r, err := NewReader(c, common.HexToAddress(proxy))
	if err != nil {
		return err
	}

	path := configfetch.FeedsPath(chain)
	configs, err := configfetch.Get[[]Config](ctx, path)
	if err != nil {
		return fmt.Errorf("fetch feed config: %w", err)
	}
	if err := r.ApplyConfigs(ctx, configs); err != nil {
		return err
	}

	reloader := configreload.New(path, len(configs), func(c Config) string { return c.Name }, r.ApplyConfigs)
	go reloader.Run(ctx, configreload.Interval(), nil)

	interval := pollInterval()
	log.Info().Str("Player", "OnchainChecker").Int("feeds", len(configs)).Dur("interval", interval).Msg("on-chain reader started")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := r.Poll(ctx); err != nil && ctx.Err() == nil {
			log.Warn().Str("Player", "OnchainChecker").Err(err).Msg("poll failed, retrying next tick")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func pollInterval() time.Duration {
	v := os.Getenv(IntervalEnv)
	if v == "" {
		return DefaultPollInterval
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		log.Warn().Str("value", v).Msg("invalid " + IntervalEnv + ", using default")
		return DefaultPollInterval
	}
	return d
}

func NewReader(c chainClient, proxy common.Address) (*Reader, error) {
	feedsABI, err := utils.GenerateViewABI("feeds", "bytes32", "address")
	if err != nil {
		return nil, err
	}
	latestRoundABI, err := utils.GenerateViewABI("latestRoundData", "", "uint64,int256,uint256")
	if err != nil {
		return nil, err
	}
	return &Reader{
		client:         c,
		proxy:          proxy,
		feedsABI:       feedsABI,
		latestRoundABI: latestRoundABI,
		byAddress:      map[common.Address]feed{},
		addrCache:      map[string]common.Address{},
	}, nil
}

// ApplyConfigs resolves feed addresses via SubmissionProxy.feeds(keccak256(name))
// and seeds metrics from latestRoundData. Feeds that are not registered on-chain
// are skipped; feeds that disappeared from the config are dropped. Resolved
// addresses are cached, so a reload only queries new names.
func (r *Reader) ApplyConfigs(ctx context.Context, configs []Config) error {
	addrs := make([]common.Address, len(configs))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(rpcConcurrency)
	for i, cfg := range configs {
		r.mu.Lock()
		cached, ok := r.addrCache[cfg.Name]
		r.mu.Unlock()
		if ok {
			addrs[i] = cached
			continue
		}
		g.Go(func() error {
			addr, err := r.resolve(gctx, cfg.Name)
			if err != nil {
				rpcErrorsCounter.WithLabelValues("feeds").Inc()
				return fmt.Errorf("resolve %s: %w", cfg.Name, err)
			}
			addrs[i] = addr
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	next := make(map[common.Address]feed, len(configs))
	for i, cfg := range configs {
		if addrs[i] == (common.Address{}) {
			log.Warn().Str("Player", "OnchainChecker").Str("feed", cfg.Name).Msg("feed not registered in SubmissionProxy, skipping")
			continue
		}
		f := feed{name: cfg.Name, address: addrs[i]}
		if cfg.SubmitInterval != nil {
			f.interval = time.Duration(*cfg.SubmitInterval) * time.Millisecond
		}
		next[addrs[i]] = f
	}

	r.mu.Lock()
	prev := r.byAddress
	r.byAddress = next
	for _, f := range next {
		r.addrCache[f.name] = f.address
	}
	r.mu.Unlock()

	for addr, f := range prev {
		if n, ok := next[addr]; !ok || n.name != f.name {
			deleteFeedMetrics(f.name)
		}
	}

	var seeds errgroup.Group
	seeds.SetLimit(rpcConcurrency)
	for addr, f := range next {
		expectedIntervalGauge.WithLabelValues(f.name).Set(f.interval.Seconds())
		if _, known := prev[addr]; known {
			continue
		}
		seeds.Go(func() error {
			if err := r.seed(ctx, f); err != nil {
				rpcErrorsCounter.WithLabelValues("latestRoundData").Inc()
				log.Warn().Str("Player", "OnchainChecker").Str("feed", f.name).Err(err).Msg("latestRoundData failed")
			}
			return nil
		})
	}
	_ = seeds.Wait()
	feedsGauge.Set(float64(len(next)))
	return nil
}

func (r *Reader) resolve(ctx context.Context, name string) (common.Address, error) {
	out, err := r.call(ctx, r.feedsABI, "feeds", r.proxy, crypto.Keccak256Hash([]byte(name)))
	if err != nil {
		return common.Address{}, err
	}
	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected feeds() output %T", out[0])
	}
	return addr, nil
}

func (r *Reader) seed(ctx context.Context, f feed) error {
	out, err := r.call(ctx, r.latestRoundABI, "latestRoundData", f.address)
	if err != nil {
		return err
	}
	answer, ok1 := out[1].(*big.Int)
	updatedAt, ok2 := out[2].(*big.Int)
	if !ok1 || !ok2 {
		return fmt.Errorf("unexpected latestRoundData output")
	}
	if updatedAt.Sign() == 0 {
		return nil // never updated
	}
	answerGauge.WithLabelValues(f.name).Set(bigToFloat(answer))
	updatedGauge.WithLabelValues(f.name).Set(float64(updatedAt.Int64()))
	return nil
}

func (r *Reader) call(ctx context.Context, a *abi.ABI, method string, to common.Address, args ...interface{}) ([]interface{}, error) {
	data, err := a.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	res, err := r.client.CallContract(ctx, kaia.CallMsg{To: &to, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	out, err := a.Unpack(method, res)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: empty output", method)
	}
	return out, nil
}

// Poll processes FeedUpdated logs from the last processed block up to head.
// On error nothing is advanced, so the same range is retried next time.
func (r *Reader) Poll(ctx context.Context) error {
	headBig, err := r.client.BlockNumber(ctx)
	if err != nil {
		rpcErrorsCounter.WithLabelValues("blockNumber").Inc()
		return err
	}
	head := headBig.Uint64()

	r.mu.Lock()
	addrs := make([]common.Address, 0, len(r.byAddress))
	for a := range r.byAddress {
		addrs = append(addrs, a)
	}
	from := r.lastBlock + 1
	if r.lastBlock == 0 {
		from = head // first poll: state already seeded by latestRoundData
	}
	r.mu.Unlock()

	if len(addrs) == 0 || from > head {
		lastPollSuccessGauge.SetToCurrentTime()
		return nil
	}
	if head-from+1 > MaxBlockRange {
		log.Warn().Str("Player", "OnchainChecker").Uint64("from", from).Uint64("head", head).Msg("block gap too large, skipping ahead")
		from = head - MaxBlockRange + 1
	}

	logs, err := r.client.FilterLogs(ctx, kaia.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(head),
		Addresses: addrs,
		Topics:    [][]common.Hash{{feedUpdatedTopic}},
	})
	if err != nil {
		rpcErrorsCounter.WithLabelValues("getLogs").Inc()
		return err
	}

	times, err := r.blockTimes(ctx, logs)
	if err != nil {
		rpcErrorsCounter.WithLabelValues("getBlockHeader").Inc()
		return err
	}

	r.mu.Lock()
	r.processLogs(logs, times)
	r.lastBlock = head
	r.mu.Unlock()

	lastBlockGauge.Set(float64(head))
	lastPollSuccessGauge.SetToCurrentTime()
	return nil
}

func (r *Reader) blockTimes(ctx context.Context, logs []types.Log) (map[uint64]int64, error) {
	times := map[uint64]int64{}
	for _, l := range logs {
		if _, ok := times[l.BlockNumber]; ok {
			continue
		}
		h, err := r.client.HeaderByNumber(ctx, new(big.Int).SetUint64(l.BlockNumber))
		if err != nil {
			return nil, err
		}
		times[l.BlockNumber] = h.Time.Int64()
	}
	return times, nil
}

// processLogs applies logs in chain order. Caller holds r.mu.
func (r *Reader) processLogs(logs []types.Log, times map[uint64]int64) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})
	for _, l := range logs {
		if l.Removed || len(l.Topics) < 2 || l.Topics[0] != feedUpdatedTopic {
			continue
		}
		f, ok := r.byAddress[l.Address]
		if !ok {
			continue
		}
		answerGauge.WithLabelValues(f.name).Set(bigToFloat(decodeAnswer(l.Topics[1])))
		updatedGauge.WithLabelValues(f.name).Set(float64(times[l.BlockNumber]))
		updatesCounter.WithLabelValues(f.name).Inc()
	}
}

// decodeAnswer reads the indexed int256 answer (two's complement topic).
func decodeAnswer(topic common.Hash) *big.Int {
	return math.S256(new(big.Int).SetBytes(topic.Bytes()))
}

func bigToFloat(x *big.Int) float64 {
	f, _ := new(big.Float).SetInt(x).Float64()
	return f
}
