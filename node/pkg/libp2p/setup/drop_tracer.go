package setup

import (
	"bisonai.com/miko/node/pkg/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// dropTracer counts outbound RPCs that pubsub drops, mostly because a peer's
// outbound queue is full (#2822). Every other event is ignored. Pubsub calls
// raw tracers synchronously from its event loop, so this must stay cheap.
type dropTracer struct{}

var _ pubsub.RawTracer = dropTracer{}

func (dropTracer) DropRPC(rpc *pubsub.RPC, _ peer.ID) {
	if rpc == nil {
		metrics.RecordP2PDroppedRPC(0, 0, 0)
		return
	}
	control := 0
	if c := rpc.GetControl(); c != nil {
		control = len(c.GetGraft()) + len(c.GetPrune()) + len(c.GetIhave()) + len(c.GetIwant()) + len(c.GetIdontwant())
	}
	metrics.RecordP2PDroppedRPC(len(rpc.GetPublish()), len(rpc.GetSubscriptions()), control)
}

func (dropTracer) AddPeer(peer.ID, protocol.ID)          {}
func (dropTracer) RemovePeer(peer.ID)                    {}
func (dropTracer) Join(string)                           {}
func (dropTracer) Leave(string)                          {}
func (dropTracer) Graft(peer.ID, string)                 {}
func (dropTracer) Prune(peer.ID, string)                 {}
func (dropTracer) ValidateMessage(*pubsub.Message)       {}
func (dropTracer) DeliverMessage(*pubsub.Message)        {}
func (dropTracer) RejectMessage(*pubsub.Message, string) {}
func (dropTracer) DuplicateMessage(*pubsub.Message)      {}
func (dropTracer) ThrottlePeer(peer.ID)                  {}
func (dropTracer) RecvRPC(*pubsub.RPC)                   {}
func (dropTracer) SendRPC(*pubsub.RPC, peer.ID)          {}
func (dropTracer) UndeliverableMessage(*pubsub.Message)  {}
