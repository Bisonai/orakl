package raft

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

// DefaultControlTopic is the shared GossipSub topic every node joins once for
// combined Raft heartbeats. It carries control-plane traffic only; per-feed data
// (trigger/priceData/priceFix/proof) stays on the per-feed topics (#2558).
const DefaultControlTopic = "orakl-raft-control"

// ControlTopicName returns the shared heartbeat control-topic name, overridable
// via the P2P_HEARTBEAT_CONTROL_TOPIC env var.
func ControlTopicName() string {
	if v := strings.TrimSpace(os.Getenv("P2P_HEARTBEAT_CONTROL_TOPIC")); v != "" {
		return v
	}
	return DefaultControlTopic
}

// HeartbeatCoordinator collapses the per-feed Raft heartbeats of the ~150 local
// Raft groups into ONE combined heartbeat per tick on the shared control topic.
//
// Send (Phase 2, P2P_HEARTBEAT_BATCH on): each tick it snapshots {feedId -> term}
// for every locally-led feed and publishes a single BatchHeartbeat, paying the
// signature/framing/packet overhead once per tick instead of ~150x.
//
// Receive (always on, so a mixed fleet is safe): on a BatchHeartbeat it fans each
// {feedId, term} out to the matching Raft group's applyHeartbeat, identical in
// effect to a per-feed heartbeat today. Feeds absent from the message fall through
// to the existing missed-heartbeat logic.
type HeartbeatCoordinator struct {
	host  host.Host
	ps    *pubsub.PubSub
	topic *pubsub.Topic

	mu    sync.RWMutex
	feeds map[int32]*Raft

	buffer chan *pubsub.Message
}

// NewHeartbeatCoordinator builds a coordinator bound to an already-joined control
// topic. Register the local Raft groups with Reset before/after starting Run.
func NewHeartbeatCoordinator(h host.Host, ps *pubsub.PubSub, topic *pubsub.Topic, messageBuffer int) *HeartbeatCoordinator {
	return &HeartbeatCoordinator{
		host:   h,
		ps:     ps,
		topic:  topic,
		feeds:  make(map[int32]*Raft),
		buffer: make(chan *pubsub.Message, messageBuffer),
	}
}

// Reset replaces the registry of feedId -> Raft. Called on startup and whenever
// the aggregator app reloads its configs, so the coordinator always tracks the
// current set of local Raft groups.
func (c *HeartbeatCoordinator) Reset(feeds map[int32]*Raft) {
	next := make(map[int32]*Raft, len(feeds))
	for id, r := range feeds {
		next[id] = r
	}
	c.mu.Lock()
	c.feeds = next
	c.mu.Unlock()
}

func (c *HeartbeatCoordinator) hostID() string {
	return c.host.ID().String()
}

// Run drives the control plane: it subscribes for incoming combined heartbeats
// and, while P2P_HEARTBEAT_BATCH is enabled, emits one combined heartbeat every
// HEARTBEAT_TIMEOUT tick. It blocks until ctx is cancelled.
func (c *HeartbeatCoordinator) Run(ctx context.Context) {
	go c.subscribe(ctx)

	ticker := time.NewTicker(HEARTBEAT_TIMEOUT)
	defer ticker.Stop()

	for {
		select {
		case rawMsg := <-c.buffer:
			go func(m *pubsub.Message) {
				defer func() {
					if rec := recover(); rec != nil {
						log.Error().Msgf("recovered from panic in heartbeat fan-out: %v", rec)
					}
				}()
				c.handleRaw(m)
			}(rawMsg)
		case <-ticker.C:
			if HeartbeatBatchEnabled() {
				c.broadcast(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *HeartbeatCoordinator) subscribe(ctx context.Context) {
	sub, err := c.topic.Subscribe()
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to control topic")
		return
	}
	defer func() {
		sub.Cancel()
		c.topic.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("control topic context cancelled")
			return
		default:
			rawMsg, err := sub.Next(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to get message from control topic")
				continue
			}
			c.buffer <- rawMsg
		}
	}
}

// broadcast snapshots the term of every locally-led feed under short per-Raft
// locks, then publishes one combined heartbeat lock-free.
func (c *HeartbeatCoordinator) broadcast(ctx context.Context) {
	c.mu.RLock()
	terms := make(map[int32]int, len(c.feeds))
	for id, r := range c.feeds {
		if term, isLeader := r.leaderTermSnapshot(); isLeader {
			terms[id] = term
		}
	}
	c.mu.RUnlock()

	if len(terms) == 0 {
		return
	}

	inner, err := encodeInner(BatchHeartbeatMessage{Terms: terms})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal batch heartbeat message")
		return
	}

	message := Message{
		Type:      BatchHeartbeat,
		SentFrom:  c.hostID(),
		Data:      inner,
		Timestamp: time.Now(),
	}
	data, err := encodeMessage(message)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal batch heartbeat")
		return
	}
	if err := c.topic.Publish(ctx, data); err != nil {
		log.Error().Err(err).Msg("failed to publish batch heartbeat")
	}
}

func (c *HeartbeatCoordinator) handleRaw(rawMsg *pubsub.Message) {
	msg, err := decodeMessage(rawMsg.Data)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal control message")
		return
	}

	// control plane carries combined heartbeats only
	if msg.Type != BatchHeartbeat {
		return
	}

	// ignore our own combined heartbeat (GossipSub echoes it back)
	if msg.SentFrom == c.hostID() {
		return
	}

	var batch BatchHeartbeatMessage
	if err := decodeInner(msg.Data, &batch); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal batch heartbeat message")
		return
	}

	c.fanout(msg.SentFrom, batch.Terms)
}

// fanout delivers each {feedId, term} to the matching local Raft group's
// heartbeat handler. Feeds absent from terms are left untouched and fall through
// to the existing missed-heartbeat logic.
func (c *HeartbeatCoordinator) fanout(leaderID string, terms map[int32]int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, term := range terms {
		r, ok := c.feeds[id]
		if !ok {
			continue
		}
		if err := r.applyHeartbeat(leaderID, term); err != nil {
			log.Error().Err(err).Int32("feedId", id).Msg("failed to apply batched heartbeat")
		}
	}
}
