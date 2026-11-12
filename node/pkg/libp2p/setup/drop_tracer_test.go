package setup

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	metric:
		for _, m := range family.GetMetric() {
			for _, l := range m.GetLabel() {
				if want, ok := labels[l.GetName()]; ok && want != l.GetValue() {
					continue metric
				}
			}
			return m.GetCounter().GetValue()
		}
	}
	return 0
}

func dropped(t *testing.T) (rpcs, publish, subs, control float64) {
	t.Helper()
	return counterValue(t, metricnames.P2PDroppedRPCs, nil),
		counterValue(t, metricnames.P2PDroppedMessages, map[string]string{metricnames.KindLabel: "publish"}),
		counterValue(t, metricnames.P2PDroppedMessages, map[string]string{metricnames.KindLabel: "subscription"}),
		counterValue(t, metricnames.P2PDroppedMessages, map[string]string{metricnames.KindLabel: "control"})
}

func TestDropTracerCountsItemsByKind(t *testing.T) {
	r0, p0, s0, c0 := dropped(t)

	topic := "t"
	rpc := &pubsub.RPC{RPC: pb.RPC{
		Publish:       []*pb.Message{{Topic: &topic}, {Topic: &topic}},
		Subscriptions: []*pb.RPC_SubOpts{{Topicid: &topic}},
		Control: &pb.ControlMessage{
			Graft: []*pb.ControlGraft{{TopicID: &topic}},
			Prune: []*pb.ControlPrune{{TopicID: &topic}},
			Ihave: []*pb.ControlIHave{{TopicID: &topic}},
		},
	}}
	dropTracer{}.DropRPC(rpc, "")
	dropTracer{}.DropRPC(nil, "")

	r1, p1, s1, c1 := dropped(t)
	require.Equal(t, 2.0, r1-r0)
	require.Equal(t, 2.0, p1-p0)
	require.Equal(t, 1.0, s1-s0)
	require.Equal(t, 3.0, c1-c0)
}

// A real overflow: the peer speaks gossipsub, announces a subscription, then never
// reads. The sender's stream blocks once TCP buffers fill, its 1-slot outbound
// queue overflows, and the tracer MakePubsub installs must count the drops.
func TestDropTracerSeesQueueOverflow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sender, err := NewHost(ctx)
	require.NoError(t, err)
	defer sender.Close()
	stuck, err := NewHost(ctx)
	require.NoError(t, err)
	defer stuck.Close()

	const topicName = "drop-tracer-test"
	stuck.SetStreamHandler(pubsub.GossipSubID_v11, func(s network.Stream) {
		// announce the subscription on our own stream, then read nothing
		out, err := stuck.NewStream(ctx, s.Conn().RemotePeer(), pubsub.GossipSubID_v11)
		if err != nil {
			return
		}
		subscribe, topic := true, topicName
		msg, err := (&pb.RPC{Subscriptions: []*pb.RPC_SubOpts{{Subscribe: &subscribe, Topicid: &topic}}}).Marshal()
		if err != nil {
			return
		}
		frame := binary.AppendUvarint(nil, uint64(len(msg)))
		_, _ = out.Write(append(frame, msg...))
		<-ctx.Done()
	})

	ps, err := pubsub.NewGossipSub(ctx, sender, pubsub.WithPeerOutboundQueueSize(1), pubsub.WithRawTracer(dropTracer{}))
	require.NoError(t, err)
	topic, err := ps.Join(topicName)
	require.NoError(t, err)
	require.NoError(t, sender.Connect(ctx, peer.AddrInfo{ID: stuck.ID(), Addrs: stuck.Addrs()}))
	require.Eventually(t, func() bool { return len(topic.ListPeers()) == 1 }, 10*time.Second, 20*time.Millisecond, "stuck peer never subscribed")

	r0, p0, _, _ := dropped(t)
	n := 0
	require.Eventually(t, func() bool {
		for i := 0; i < 50; i++ {
			// pubsub keeps the slice after Publish returns: one buffer per message
			payload := make([]byte, 64*1024)
			binary.BigEndian.PutUint32(payload, uint32(n))
			n++
			_ = topic.Publish(ctx, payload)
		}
		r1, p1, _, _ := dropped(t)
		return r1 > r0 && p1 > p0
	}, 15*time.Second, 10*time.Millisecond, "no dropped RPC counted towards a peer that stopped reading")
}

func TestPubsubOptionsValid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := NewHost(ctx)
	require.NoError(t, err)
	defer h.Close()
	_, err = MakePubsub(ctx, h)
	require.NoError(t, err)
	require.Len(t, pubsubOptions(), 3)
}
