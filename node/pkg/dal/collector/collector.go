package collector

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/chain/websocketchainreader"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/common/types"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/db"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	DefaultDecimals = "8"
	GetAllOracles   = "getAllOracles() public view returns (address[] memory)"
	OracleAdded     = "OracleAdded(address oracle, uint256 expirationTime)"
)

type Collector struct {
	OutgoingStream   map[int32]chan *dalcommon.OutgoingSubmissionData
	Symbols          map[int32]string
	FeedHashes       map[int32][]byte
	LatestTimestamps map[int32]time.Time
	LatestData       map[string]*dalcommon.OutgoingSubmissionData
	CachedWhitelist  []klaytncommon.Address

	baseRediscribe *db.Rediscribe
	subRediscribe  *db.Rediscribe

	IsRunning  bool
	CancelFunc context.CancelFunc

	chainReader                 *websocketchainreader.ChainReader
	submissionProxyContractAddr string

	mu sync.RWMutex
}

func NewCollector(ctx context.Context, configs []types.Config) (*Collector, error) {
	kaiaWebsocketUrl := os.Getenv("KAIA_WEBSOCKET_URL")
	if kaiaWebsocketUrl == "" {
		return nil, errors.New("KAIA_WEBSOCKET_URL is not set")
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

	chainReader, err := websocketchainreader.New(websocketchainreader.WithKaiaWebsocketUrl(kaiaWebsocketUrl))
	if err != nil {
		return nil, err
	}

	initialWhitelist, err := getAllOracles(ctx, chainReader, submissionProxyContractAddr)
	if err != nil {
		return nil, err
	}

	collector := &Collector{
		OutgoingStream:              make(map[int32]chan *dalcommon.OutgoingSubmissionData, len(configs)),
		Symbols:                     make(map[int32]string, len(configs)),
		FeedHashes:                  make(map[int32][]byte, len(configs)),
		LatestTimestamps:            make(map[int32]time.Time),
		LatestData:                  make(map[string]*dalcommon.OutgoingSubmissionData),
		chainReader:                 chainReader,
		CachedWhitelist:             initialWhitelist,
		submissionProxyContractAddr: submissionProxyContractAddr,
	}

	redisTopics := []string{}
	for _, config := range configs {
		collector.OutgoingStream[config.ID] = make(chan *dalcommon.OutgoingSubmissionData, 1000)
		collector.Symbols[config.ID] = config.Name
		collector.FeedHashes[config.ID] = crypto.Keccak256([]byte(config.Name))
		redisTopics = append(redisTopics, keys.SubmissionDataStreamKey(config.ID))
	}

	baseRediscribe, err := db.NewRediscribe(ctx, db.WithRedisHost(baseRedisHost), db.WithRedisPort(baseRedisPort), db.WithRedisTopics(redisTopics), db.WithRedisRouter(collector.redisRouter))
	if err != nil {
		return nil, err
	}
	collector.baseRediscribe = baseRediscribe

	if subRedisHost != "" && subRedisPort != "" {
		subRediscribe, err := db.NewRediscribe(ctx, db.WithRedisHost(subRedisHost), db.WithRedisPort(subRedisPort), db.WithRedisTopics(redisTopics), db.WithRedisRouter(collector.redisRouter))
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
	c.trackOracleAdded(ctxWithCancel)
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
	result := make([]dalcommon.OutgoingSubmissionData, 0, len(c.Symbols))
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

	old, ok := c.LatestTimestamps[data.GlobalAggregate.ConfigID]
	if !ok || !old.After(data.GlobalAggregate.Timestamp) {
		c.LatestTimestamps[data.GlobalAggregate.ConfigID] = data.GlobalAggregate.Timestamp
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
			log.Debug().Str("Player", "DalCollector").Str("Symbol", c.Symbols[data.GlobalAggregate.ConfigID]).Msg("old data recieved")
			return
		}

		result, err := c.IncomingDataToOutgoingData(ctx, data)
		if err != nil {
			log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to convert incoming data to outgoing data")
			return
		}

		defer func(data *dalcommon.OutgoingSubmissionData) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.LatestData[data.Symbol] = data
		}(result)
		c.OutgoingStream[data.GlobalAggregate.ConfigID] <- result
	}
}

func (c *Collector) IncomingDataToOutgoingData(ctx context.Context, data *aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error) {
	c.mu.RLock()
	whitelist := c.CachedWhitelist
	c.mu.RUnlock()
	orderedProof, err := orderProof(
		ctx,
		data.Proof.Proof,
		data.GlobalAggregate.Value,
		data.GlobalAggregate.Timestamp,
		c.Symbols[data.GlobalAggregate.ConfigID],
		whitelist)
	if err != nil {
		log.Error().Err(err).Str("Player", "DalCollector").Str("Symbol", c.Symbols[data.GlobalAggregate.ConfigID]).Msg("failed to order proof")
		if errors.Is(err, errorsentinel.ErrDalSignerNotWhitelisted) {
			go func(ctx context.Context, chainHelper *websocketchainreader.ChainReader, contractAddress string) {
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
	return &dalcommon.OutgoingSubmissionData{
		Symbol:        c.Symbols[data.GlobalAggregate.ConfigID],
		Value:         strconv.FormatInt(data.GlobalAggregate.Value, 10),
		AggregateTime: strconv.FormatInt(data.GlobalAggregate.Timestamp.UnixMilli(), 10),
		Proof:         formatBytesToHex(orderedProof),
		FeedHash:      formatBytesToHex(c.FeedHashes[data.GlobalAggregate.ConfigID]),
		Decimals:      DefaultDecimals,
	}, nil
}

func (c *Collector) trackOracleAdded(ctx context.Context) {
	eventTriggered := make(chan any)
	err := subscribeAddOracleEvent(ctx, c.chainReader, c.submissionProxyContractAddr, eventTriggered)
	if err != nil {
		log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to subscribe to oracle added event")
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-eventTriggered:
				newList, err := getAllOracles(ctx, c.chainReader, c.submissionProxyContractAddr)
				if err != nil {
					log.Error().Err(err).Str("Player", "DalCollector").Msg("failed to get all oracles")
				}
				c.mu.Lock()
				c.CachedWhitelist = newList
				c.mu.Unlock()
			}
		}
	}()
}
