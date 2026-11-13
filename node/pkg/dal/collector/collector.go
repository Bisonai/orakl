package collector

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/chain/chainreader"
	"bisonai.com/miko/node/pkg/chain/proof"
	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/common/types"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/db"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/utils/pool"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	WhitelistRefreshInterval = 10 * time.Second
	DefaultDecimals          = "8"
	GetAllOracles            = "getAllOracles() public view returns (address[] memory)"
	OracleAdded              = "OracleAdded(address oracle, uint256 expirationTime)"
	ProcessWorkerCount       = 200
	// OnDemandRefreshCooldown bounds how often a proof rejection may trigger an immediate
	// getAllOracles read. The path is only an accelerator over the WhitelistRefreshInterval
	// ticker, so at most one on-demand refresh per cooldown is enough to recover quickly from a
	// legitimate rotation while preventing a sustained mismatch (e.g. a signer freeze, where every
	// feed's proof is rejected ~2.5x/s) from spawning hundreds of on-chain calls per second.
	OnDemandRefreshCooldown = 1 * time.Second
	OutgoingStreamSize      = 1000
)

// ResubscribeGrace keeps the previous redis subscriptions open this long after a config reload
// started new ones (which connect asynchronously), so unchanged symbols see no gap (duplicates
// are dropped by the timestamp compare-and-swap). A var so tests can shorten it.
var ResubscribeGrace = 30 * time.Second

type Config = types.Config

type Collector struct {
	OutgoingStream map[string]chan *dalcommon.OutgoingSubmissionData

	FeedHashes       map[string][]byte
	LatestTimestamps map[string]time.Time
	LatestData       map[string]*dalcommon.OutgoingSubmissionData
	Configs          map[string]Config
	CachedWhitelist  []kaiacommon.Address

	baseRediscribe *db.Rediscribe
	subRediscribe  *db.Rediscribe

	baseRedisHost, baseRedisPort string
	subRedisHost, subRedisPort   string

	// processCtx is the collector's run context, set in Start before any
	// subscription runs (read without lock on the hot path).
	processCtx context.Context

	// redisMu guards runCtx and redisCancel (the live subscriptions, replaced on reload).
	redisMu     sync.Mutex
	runCtx      context.Context
	redisCancel context.CancelFunc
	// subscribe starts subscriptions to topics under ctx and returns a channel
	// closed once they are connected; replaceable in tests.
	subscribe func(ctx context.Context, topics []string) (<-chan struct{}, error)
	// toOutgoing verifies and converts incoming data; nil means IncomingDataToOutgoingData.
	// Replaceable in tests to control verification timing.
	toOutgoing func(ctx context.Context, data *aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error)

	IsRunning  bool
	CancelFunc context.CancelFunc

	chainReader                 *chainreader.ChainReader
	submissionProxyContractAddr string

	processPool *pool.Pool
	mu          sync.RWMutex

	// lastOnDemandRefresh is the UnixNano of the last rejection-triggered whitelist refresh; it
	// doubles as the rate-limit claim so concurrent rejections fire at most one refresh per
	// OnDemandRefreshCooldown (see maybeRefreshWhitelist).
	lastOnDemandRefresh atomic.Int64
}

func NewCollector(ctx context.Context, configs []types.Config) (*Collector, error) {
	kaiaRestUrl := os.Getenv("KAIA_PROVIDER_URL")
	if kaiaRestUrl == "" {
		return nil, errors.New("KAIA_PROVIDER_URL is not set")
	}

	submissionProxyContractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if submissionProxyContractAddr == "" {
		return nil, errors.New("SUBMISSION_PROXY_CONTRACT is not set")
	}

	baseRedisHost := os.Getenv("REDIS_HOST")
	if baseRedisHost == "" {
		return nil, errors.New("REDIS_HOST is not set")
	}

	baseRedisPort := os.Getenv("REDIS_PORT")
	if baseRedisPort == "" {
		return nil, errors.New("REDIS_PORT is not set")
	}

	subRedisHost := os.Getenv("SUB_REDIS_HOST")
	if subRedisHost == "" {
		log.Warn().Msg("sub redis host not set")
	}

	subRedisPort := os.Getenv("SUB_REDIS_PORT")
	if subRedisPort == "" {
		log.Warn().Msg("sub redis port not set")
	}

	chainReader, err := chainreader.NewChainReader(kaiaRestUrl)
	if err != nil {
		return nil, err
	}

	initialWhitelist, err := getAllOracles(ctx, chainReader, submissionProxyContractAddr)
	if err != nil {
		return nil, err
	}

	collector := &Collector{
		OutgoingStream:   make(map[string]chan *dalcommon.OutgoingSubmissionData, len(configs)),
		FeedHashes:       make(map[string][]byte, len(configs)),
		LatestTimestamps: make(map[string]time.Time),
		LatestData:       make(map[string]*dalcommon.OutgoingSubmissionData),
		Configs:          make(map[string]Config, len(configs)),

		baseRedisHost: baseRedisHost,
		baseRedisPort: baseRedisPort,
		subRedisHost:  subRedisHost,
		subRedisPort:  subRedisPort,

		chainReader:                 chainReader,
		CachedWhitelist:             initialWhitelist,
		submissionProxyContractAddr: submissionProxyContractAddr,
		processPool:                 pool.NewPool(ProcessWorkerCount),
	}
	collector.subscribe = collector.redisSubscribe

	redisTopics := []string{}
	for _, config := range configs {
		collector.OutgoingStream[config.Name] = make(chan *dalcommon.OutgoingSubmissionData, OutgoingStreamSize)
		collector.FeedHashes[config.Name] = crypto.Keccak256([]byte(config.Name))
		collector.Configs[config.Name] = config
		redisTopics = append(redisTopics, keys.SubmissionDataStreamKey(config.Name))
	}

	baseRediscribe, err := db.NewRediscribe(
		ctx,
		db.WithRedisHost(baseRedisHost),
		db.WithRedisPort(baseRedisPort),
		db.WithRedisTopics(redisTopics),
		db.WithRedisRouter(collector.redisRouter))
	if err != nil {
		return nil, err
	}
	collector.baseRediscribe = baseRediscribe

	if subRedisHost != "" && subRedisPort != "" {
		subRediscribe, err := db.NewRediscribe(
			ctx,
			db.WithRedisHost(subRedisHost), db.WithRedisPort(subRedisPort), db.WithRedisTopics(redisTopics), db.WithRedisRouter(collector.redisRouter))
		if err != nil {
			return nil, err
		}
		collector.subRediscribe = subRediscribe
	}

	return collector, nil
}

func (c *Collector) Start(ctx context.Context) {
	if c.IsRunning {
		log.Warn().Str("Player", "DalCollector").Msg("Collector already running, skipping start")
		return
	}
	c.IsRunning = true

	ctxWithCancel, cancel := context.WithCancel(ctx)
	c.CancelFunc = cancel
	c.IsRunning = true

	redisCtx, redisCancel := context.WithCancel(ctxWithCancel)
	c.processCtx = ctxWithCancel
	c.redisMu.Lock()
	c.runCtx, c.redisCancel = ctxWithCancel, redisCancel
	c.redisMu.Unlock()

	c.processPool.Run(ctxWithCancel)
	c.receive(redisCtx)
	go c.refreshOracles(ctxWithCancel)
}

func (c *Collector) GetLatestData(symbol string) (*dalcommon.OutgoingSubmissionData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, ok := c.LatestData[symbol]
	if !ok {
		return nil, errors.New("symbol not found")
	}
	return result, nil
}

func (c *Collector) GetAllLatestData() []dalcommon.OutgoingSubmissionData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]dalcommon.OutgoingSubmissionData, 0, len(c.FeedHashes))
	for _, value := range c.LatestData {
		result = append(result, *value)
	}

	return result
}

func (c *Collector) Stop() {
	if c.CancelFunc != nil {
		c.CancelFunc()
		c.IsRunning = false
	}
}

func (c *Collector) receive(ctx context.Context) {
	go c.baseRediscribe.Start(ctx)

	if c.subRediscribe != nil {
		// wait for sidecar to be ready
		time.Sleep(10 * time.Second)
		go c.subRediscribe.Start(ctx)
	}
}

func (c *Collector) redisRouter(ctx context.Context, msg *redis.Message) error {
	var data *aggregator.SubmissionData
	err := json.Unmarshal([]byte(msg.Payload), &data)
	if err != nil {
		return err
	}

	// Process under the collector's context, not the subscription's: a
	// config reload cancels old subscriptions, which must not abort queued
	// jobs or an on-demand whitelist refresh they start.
	if c.processCtx != nil {
		ctx = c.processCtx
	}

	c.processPool.AddJob(func() {
		c.processIncomingData(ctx, data)
	})
	return nil
}

func (c *Collector) compareAndSwapLatestTimestamp(data *aggregator.SubmissionData) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// removed by a config reload (old subscriptions linger for ResubscribeGrace)
	if _, known := c.FeedHashes[data.Symbol]; !known {
		return false
	}

	old, ok := c.LatestTimestamps[data.Symbol]
	if !ok || data.GlobalAggregate.Timestamp.After(old) {
		c.LatestTimestamps[data.Symbol] = data.GlobalAggregate.Timestamp
		return true
	}

	return false
}

func (c *Collector) processIncomingData(ctx context.Context, data *aggregator.SubmissionData) {
	select {
	case <-ctx.Done():
		return
	default:
		valid := c.compareAndSwapLatestTimestamp(data)
		if !valid {
			log.Debug().Str("Player", "DalCollector").Str("Symbol", data.Symbol).Msg("old data recieved")
			return
		}

		toOutgoing := c.toOutgoing
		if toOutgoing == nil {
			toOutgoing = c.IncomingDataToOutgoingData
		}
		result, err := toOutgoing(ctx, data)
		if err != nil {
			log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to convert incoming data to outgoing data")
			return
		}

		// Under the lock so a concurrent RemoveSymbols neither races the map nor closes the
		// stream mid-send (the send never blocks).
		c.mu.Lock()
		defer c.mu.Unlock()
		stream, known := c.OutgoingStream[result.Symbol]
		if !known {
			return
		}
		c.LatestData[result.Symbol] = result

		select {
		case stream <- result:
		default:
			log.Debug().Str("Player", "DalCollector").Str("Symbol", result.Symbol).Msg("outgoing stream full, dropping data")
		}
	}
}

func (c *Collector) IncomingDataToOutgoingData(ctx context.Context, data *aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error) {
	c.mu.RLock()
	whitelist := make([]kaiacommon.Address, len(c.CachedWhitelist))
	copy(whitelist, c.CachedWhitelist)
	feedHashBytes, ok := c.FeedHashes[data.Symbol]
	config, hasConfig := c.Configs[data.Symbol]
	c.mu.RUnlock()

	if !ok {
		return nil, errorsentinel.ErrDalFeedHashNotFound
	}

	orderedProof, err := proof.Order(
		data.Proof.Proof,
		data.GlobalAggregate.Value,
		data.GlobalAggregate.Timestamp,
		data.Symbol,
		whitelist)
	if err != nil {
		log.Error().Err(err).Str("Player", "DalCollector").Str("Symbol", data.Symbol).Msg("failed to order proof")
		if errors.Is(err, errorsentinel.ErrDalSignerNotWhitelisted) {
			c.maybeRefreshWhitelist(ctx)
		}
		return nil, err
	}

	decimals := DefaultDecimals
	if hasConfig && config.Decimals != nil && *config.Decimals != 0 {
		decimals = strconv.Itoa(*config.Decimals)
	}

	return &dalcommon.OutgoingSubmissionData{
		Symbol:        data.Symbol,
		Value:         strconv.FormatInt(data.GlobalAggregate.Value, 10),
		AggregateTime: strconv.FormatInt(data.GlobalAggregate.Timestamp.UnixMilli(), 10),
		Proof:         formatBytesToHex(orderedProof),
		FeedHash:      formatBytesToHex(feedHashBytes),
		Decimals:      decimals,
	}, nil
}

// maybeRefreshWhitelist triggers an immediate getAllOracles read after a proof was rejected for a
// non-whitelisted signer: the local cache may simply be lagging a legitimate oracle rotation, and
// this recovers faster than waiting for the WhitelistRefreshInterval ticker. It is rate-limited to
// at most one refresh per OnDemandRefreshCooldown — a single atomic timestamp both enforces the
// cooldown and claims the slot, so a flood of concurrent rejections (as in a signer freeze where
// every feed's proof is rejected) collapses to one on-chain call per cooldown instead of one per
// rejected message.
func (c *Collector) maybeRefreshWhitelist(ctx context.Context) {
	if !c.claimRefreshSlot() {
		return
	}

	go func() {
		newList, err := getAllOracles(ctx, c.chainReader, c.submissionProxyContractAddr)
		if err != nil {
			log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to refresh oracles on-demand")
			return
		}
		c.mu.Lock()
		c.CachedWhitelist = newList
		c.mu.Unlock()
	}()
}

// claimRefreshSlot reports whether the caller may fire an on-demand whitelist refresh now,
// enforcing OnDemandRefreshCooldown.
func (c *Collector) claimRefreshSlot() bool {
	return c.claimRefreshSlotAt(time.Now().UnixNano())
}

// claimRefreshSlotAt is claimRefreshSlot with an injected clock (UnixNano) so the cooldown can be
// exercised deterministically in tests. The single atomic doubles as the claim: among many callers
// within one cooldown window, exactly one CompareAndSwap succeeds, so exactly one gets true.
func (c *Collector) claimRefreshSlotAt(now int64) bool {
	last := c.lastOnDemandRefresh.Load()
	if now-last < int64(OnDemandRefreshCooldown) {
		return false
	}
	return c.lastOnDemandRefresh.CompareAndSwap(last, now)
}

func (c *Collector) refreshOracles(ctx context.Context) {
	ticker := time.NewTicker(WhitelistRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newOracles, err := getAllOracles(ctx, c.chainReader, c.submissionProxyContractAddr)
			if err != nil {
				log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to get all oracles")
			} else {
				c.mu.Lock()
				c.CachedWhitelist = newOracles
				c.mu.Unlock()
			}
		}
	}
}
