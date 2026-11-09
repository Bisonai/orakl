package raft

import (
	"context"
	"testing"
	"time"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metricFamily reads a family from the registry /metrics is served from, and
// fails the test if it has no series.
func metricFamily(t *testing.T, name string) *dto.MetricFamily {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("%s has no series", name)
	return nil
}

// metricSeries returns the series with the given label, and fails the test if it
// doesn't exist.
func metricSeries(t *testing.T, name, labelName, labelValue string) *dto.Metric {
	t.Helper()
	for _, metric := range metricFamily(t, name).GetMetric() {
		for _, label := range metric.GetLabel() {
			if label.GetName() == labelName && label.GetValue() == labelValue {
				return metric
			}
		}
	}
	t.Fatalf("%s{%s=%q} has no series", name, labelName, labelValue)
	return nil
}

func counterValue(t *testing.T, name string, msgType MessageType) float64 {
	t.Helper()
	return metricSeries(t, name, metricnames.TypeLabel, string(msgType)).GetCounter().GetValue()
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
				// a type's series only exists after its first publish
				require.NoError(t, node.PublishMessage(ctx, Message{Type: msgType, SentFrom: node.GetHostId(), Data: []byte(`{}`)}))
				_, err := sub.Next(ctx)
				require.NoError(t, err)

				messages :=counterValue(t, metricnames.P2PPublishedMessages, msgType)
				size := counterValue(t, metricnames.P2PPublishedBytes, msgType)

				require.NoError(t, node.PublishMessage(ctx, Message{Type: msgType, SentFrom: node.GetHostId(), Data: []byte(`{}`)}))

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

	// a success first, so the series exist
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
