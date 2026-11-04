package raft

import (
	"context"
	"testing"
	"time"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// counterValue reads a counter from the registry /metrics is served from.
func counterValue(t *testing.T, name string, msgType MessageType) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == metricnames.TypeLabel && label.GetValue() == string(msgType) {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func newPublishTestNode(ctx context.Context, t *testing.T) *Raft {
	t.Helper()
	h, err := libp2psetup.NewHost(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })

	ps, err := libp2psetup.MakePubsub(ctx, h)
	require.NoError(t, err)

	topic, err := ps.Join(topicString)
	require.NoError(t, err)

	return NewRaftNode(h, ps, topic, 100, time.Second)
}

func TestPublishMessageCountsByType(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			t.Setenv("P2P_ENCODING", encoding)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			node := newPublishTestNode(ctx, t)
			sub, err := node.Topic.Subscribe()
			require.NoError(t, err)
			defer sub.Cancel()

			for msgType := range typeToInt {
				messages := counterValue(t, metricnames.P2PPublishedMessages, msgType)
				size := counterValue(t, metricnames.P2PPublishedBytes, msgType)

				err := node.PublishMessage(ctx, Message{Type: msgType, SentFrom: node.GetHostId(), Data: []byte(`{}`)})
				require.NoError(t, err)

				published, err := sub.Next(ctx)
				require.NoError(t, err)

				assert.Equal(t, messages+1, counterValue(t, metricnames.P2PPublishedMessages, msgType), msgType)
				assert.Equal(t, size+float64(len(published.Data)), counterValue(t, metricnames.P2PPublishedBytes, msgType), msgType)
			}
		})
	}
}

func TestPublishMessageFailureNotCounted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node := newPublishTestNode(ctx, t)
	msg := Message{Type: Heartbeat, SentFrom: node.GetHostId(), Data: []byte(`{}`)}

	// a success first, so the series exist and a wrong name can't pass as 0 == 0
	require.NoError(t, node.PublishMessage(ctx, msg))
	messages := counterValue(t, metricnames.P2PPublishedMessages, Heartbeat)
	size := counterValue(t, metricnames.P2PPublishedBytes, Heartbeat)
	require.Positive(t, messages)
	require.Positive(t, size)

	require.NoError(t, node.Topic.Close())
	require.Error(t, node.PublishMessage(ctx, msg))

	assert.Equal(t, messages, counterValue(t, metricnames.P2PPublishedMessages, Heartbeat))
	assert.Equal(t, size, counterValue(t, metricnames.P2PPublishedBytes, Heartbeat))
}
