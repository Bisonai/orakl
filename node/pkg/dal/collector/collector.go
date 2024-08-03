package collector

import (
	"context"
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
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/rs/zerolog/log"
)

const (
	DefaultDecimals = "8"
	GetAllOracles   = "getAllOracles() public view returns (address[] memory)"
	OracleAdded     = "OracleAdded(address oracle, uint256 expirationTime)"
)

type Collector struct {
	IncomingStream   map[int32]chan aggregator.SubmissionData
	OutgoingStream   map[int32]chan *dalcommon.OutgoingSubmissionData
	Symbols          map[int32]string
	FeedHashes       map[int32][]byte
	LatestTimestamps map[int32]time.Time
	LatestData       map[string]*dalcommon.OutgoingSubmissionData
	CachedWhitelist  []klaytncommon.Address

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

	chainReader, err := websocketchainreader.New(websocketchainreader.WithKaiaWebsocketUrl(kaiaWebsocketUrl))
	if err != nil {
		return nil, err
	}

	initialWhitelist, err := getAllOracles(ctx, chainReader, submissionProxyContractAddr)
	if err != nil {
		return nil, err
	}

	collector := &Collector{
		IncomingStream:              make(map[int32]chan aggregator.SubmissionData, len(configs)),
		OutgoingStream:              make(map[int32]chan *dalcommon.OutgoingSubmissionData, len(configs)),
		Symbols:                     make(map[int32]string, len(configs)),
		FeedHashes:                  make(map[int32][]byte, len(configs)),
		LatestTimestamps:            make(map[int32]time.Time),
		LatestData:                  make(map[string]*dalcommon.OutgoingSubmissionData),
		chainReader:                 chainReader,
		CachedWhitelist:             initialWhitelist,
		submissionProxyContractAddr: submissionProxyContractAddr,
	}

	for _, config := range configs {
		collector.IncomingStream[config.ID] = make(chan aggregator.SubmissionData, 1000)
		collector.OutgoingStream[config.ID] = make(chan *dalcommon.OutgoingSubmissionData, 1000)
		collector.Symbols[config.ID] = config.Name
		collector.FeedHashes[config.ID] = crypto.Keccak256([]byte(config.Name))
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
	for id := range c.IncomingStream {
		go func(id int32) {
			if err := c.receiveEach(ctx, id); err != nil {
				log.Error().Err(err).Str("Player", "DalCollector").Msg("Error in receiveEach goroutine")
			}
		}(id)
	}
}

func (c *Collector) receiveEach(ctx context.Context, configId int32) error {
	err := db.Subscribe(ctx, keys.SubmissionDataStreamKey(configId), c.IncomingStream[configId])
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-c.IncomingStream[configId]:
			go c.processIncomingData(ctx, data)
		}
	}
}

func (c *Collector) compareAndSwapLatestTimestamp(data aggregator.SubmissionData) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	old, ok := c.LatestTimestamps[data.GlobalAggregate.ConfigID]
	if !ok || !old.After(data.GlobalAggregate.Timestamp) {
		c.LatestTimestamps[data.GlobalAggregate.ConfigID] = data.GlobalAggregate.Timestamp
		return true
	}

	return false
}

func (c *Collector) processIncomingData(ctx context.Context, data aggregator.SubmissionData) {
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

func (c *Collector) IncomingDataToOutgoingData(ctx context.Context, data aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error) {
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
		if errors.Is(err, errorSentinel.ErrReporterSignerNotWhitelisted) {
			newList, getAllOraclesErr := getAllOracles(ctx, c.chainReader, c.submissionProxyContractAddr)
			if getAllOraclesErr != nil {
				log.Error().Err(getAllOraclesErr).Str("Player", "DalCollector").Msg("failed to refresh oracles")
				return nil, getAllOraclesErr
			}
			c.mu.Lock()
			c.CachedWhitelist = newList
			c.mu.Unlock()
		}
		return nil, err
	}
	return &dalcommon.OutgoingSubmissionData{
		Symbol:        c.Symbols[data.GlobalAggregate.ConfigID],
		Value:         strconv.FormatInt(data.GlobalAggregate.Value, 10),
		AggregateTime: strconv.FormatInt(data.GlobalAggregate.Timestamp.Unix(), 10),
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
