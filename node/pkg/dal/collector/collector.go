package collector

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/chain/chainreader"
	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/common/types"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/db"
	errorsentinel "bisonai.com/miko/node/pkg/error"
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
)

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

	IsRunning  bool
	CancelFunc context.CancelFunc

	chainReader                 *chainreader.ChainReader
	submissionProxyContractAddr string

	mu sync.RWMutex
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

		chainReader:                 chainReader,
		CachedWhitelist:             initialWhitelist,
		submissionProxyContractAddr: submissionProxyContractAddr,
	}

	redisTopics := []string{}
	for _, config := range configs {
		collector.OutgoingStream[config.Name] = make(chan *dalcommon.OutgoingSubmissionData, 1000)
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

	c.receive(ctxWithCancel)
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

	go c.processIncomingData(ctx, data)
	return nil
}

func (c *Collector) compareAndSwapLatestTimestamp(data *aggregator.SubmissionData) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

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

		result, err := c.IncomingDataToOutgoingData(ctx, data)
		if err != nil {
			log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to convert incoming data to outgoing data")
			return
		}

		defer func(result *dalcommon.OutgoingSubmissionData) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.LatestData[result.Symbol] = result
		}(result)
		c.OutgoingStream[result.Symbol] <- result
	}
}

func (c *Collector) IncomingDataToOutgoingData(ctx context.Context, data *aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error) {
	c.mu.RLock()
	whitelist := make([]kaiacommon.Address, len(c.CachedWhitelist))
	copy(whitelist, c.CachedWhitelist)
	c.mu.RUnlock()

	feedHashBytes, ok := c.FeedHashes[data.Symbol]
	if !ok {
		return nil, errorsentinel.ErrDalFeedHashNotFound
	}

	orderedProof, err := orderProof(
		ctx,
		data.Proof.Proof,
		data.GlobalAggregate.Value,
		data.GlobalAggregate.Timestamp,
		data.Symbol,
		whitelist)
	if err != nil {
		log.Error().Err(err).Str("Player", "DalCollector").Str("Symbol", data.Symbol).Msg("failed to order proof")
		if errors.Is(err, errorsentinel.ErrDalSignerNotWhitelisted) {
			go func(ctx context.Context, chainHelper *chainreader.ChainReader, contractAddress string) {
				newList, getAllOraclesErr := getAllOracles(ctx, chainHelper, contractAddress)
				if getAllOraclesErr != nil {
					log.Error().Err(getAllOraclesErr).Str("Player", "DalCollector").Msg("failed to refresh oracles")
					return
				}
				c.mu.Lock()
				c.CachedWhitelist = newList
				c.mu.Unlock()
			}(ctx, c.chainReader, c.submissionProxyContractAddr)
		}
		return nil, err
	}

	decimals := DefaultDecimals
	if config, ok := c.Configs[data.Symbol]; ok {
		if config.Decimals != nil && *config.Decimals != 0 {
			decimals = strconv.Itoa(*config.Decimals)
		}
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
