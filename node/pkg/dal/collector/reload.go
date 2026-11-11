package collector

import (
	"context"
	"errors"
	"sort"
	"time"

	"bisonai.com/miko/node/pkg/common/keys"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/db"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

// Streams returns a copy of the per-symbol outgoing streams.
func (c *Collector) Streams() map[string]chan *dalcommon.OutgoingSubmissionData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]chan *dalcommon.OutgoingSubmissionData, len(c.OutgoingStream))
	for symbol, stream := range c.OutgoingStream {
		out[symbol] = stream
	}
	return out
}

// UpdateConfigs applies a reloaded config bundle: it refreshes the config of
// kept symbols (e.g. decimals), adds new symbols and, if the symbol set
// changed, moves the redis subscriptions to the new set. It returns the
// streams of the added symbols and the names of the removed ones; removed
// symbols keep their state until RemoveSymbols, so the caller can first stop
// serving them (hub) and then drop them.
func (c *Collector) UpdateConfigs(ctx context.Context, configs []Config) (map[string]chan *dalcommon.OutgoingSubmissionData, []string, error) {
	if len(configs) == 0 {
		return nil, nil, errors.New("empty config bundle")
	}
	wanted := make(map[string]Config, len(configs))
	for _, config := range configs {
		wanted[config.Name] = config
	}

	c.mu.RLock()
	var added []Config
	for name, config := range wanted {
		if _, ok := c.FeedHashes[name]; !ok {
			added = append(added, config)
		}
	}
	var removed []string
	for name := range c.FeedHashes {
		if _, ok := wanted[name]; !ok {
			removed = append(removed, name)
		}
	}
	c.mu.RUnlock()
	sort.Slice(added, func(i, j int) bool { return added[i].Name < added[j].Name })
	sort.Strings(removed)

	// Add maps before subscribing so the first update of an added symbol is
	// not dropped as unknown; roll back if subscribing fails (the reload is
	// then retried and sees the same additions).
	c.mu.Lock()
	for name, config := range wanted {
		if _, ok := c.FeedHashes[name]; ok {
			c.Configs[name] = config
		}
	}
	addedStreams := make(map[string]chan *dalcommon.OutgoingSubmissionData, len(added))
	for _, config := range added {
		stream := make(chan *dalcommon.OutgoingSubmissionData, OutgoingStreamSize)
		c.OutgoingStream[config.Name] = stream
		c.FeedHashes[config.Name] = crypto.Keccak256([]byte(config.Name))
		c.Configs[config.Name] = config
		addedStreams[config.Name] = stream
	}
	c.mu.Unlock()

	if len(added) > 0 || len(removed) > 0 {
		topics := make([]string, 0, len(wanted))
		for name := range wanted {
			topics = append(topics, keys.SubmissionDataStreamKey(name))
		}
		sort.Strings(topics)
		if err := c.resubscribe(topics); err != nil {
			c.RemoveSymbols(configNames(added))
			return nil, nil, err
		}
	}
	return addedStreams, removed, nil
}

func configNames(configs []Config) []string {
	out := make([]string, len(configs))
	for i, c := range configs {
		out[i] = c.Name
	}
	return out
}

// RemoveSymbols drops the symbols' state and closes their outgoing streams
// (ending the hub's broadcaster for each). Later data for them is ignored.
func (c *Collector) RemoveSymbols(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, symbol := range symbols {
		if stream, ok := c.OutgoingStream[symbol]; ok {
			close(stream)
		}
		delete(c.OutgoingStream, symbol)
		delete(c.FeedHashes, symbol)
		delete(c.Configs, symbol)
		delete(c.LatestData, symbol)
		delete(c.LatestTimestamps, symbol)
	}
}

// resubscribe starts subscriptions to topics and closes the previous ones
// after ResubscribeGrace.
func (c *Collector) resubscribe(topics []string) error {
	c.redisMu.Lock()
	defer c.redisMu.Unlock()
	if c.runCtx == nil {
		return errors.New("collector not started")
	}
	ctx, cancel := context.WithCancel(c.runCtx)
	if err := c.subscribe(ctx, topics); err != nil {
		cancel()
		return err
	}
	if old := c.redisCancel; old != nil {
		time.AfterFunc(ResubscribeGrace, old)
	}
	c.redisCancel = cancel
	return nil
}

// redisSubscribe subscribes to topics on the base redis and, if set, the sub
// redis, until ctx is done.
func (c *Collector) redisSubscribe(ctx context.Context, topics []string) error {
	base, err := db.NewRediscribe(ctx,
		db.WithRedisHost(c.baseRedisHost), db.WithRedisPort(c.baseRedisPort),
		db.WithRedisTopics(topics), db.WithRedisRouter(c.redisRouter))
	if err != nil {
		return err
	}
	var sub *db.Rediscribe
	if c.subRedisHost != "" && c.subRedisPort != "" {
		sub, err = db.NewRediscribe(ctx,
			db.WithRedisHost(c.subRedisHost), db.WithRedisPort(c.subRedisPort),
			db.WithRedisTopics(topics), db.WithRedisRouter(c.redisRouter))
		if err != nil {
			return err
		}
	}
	go base.Start(ctx)
	if sub != nil {
		go sub.Start(ctx)
	}
	log.Info().Str("Player", "DalCollector").Int("topics", len(topics)).Msg("redis subscriptions replaced")
	return nil
}
