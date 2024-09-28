//nolint:all
package test

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/common"
	"bisonai.com/miko/node/pkg/dal/hub"
	wsfcommon "bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestCollectorStartAndStop(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	collector := testItems.Collector
	assert.True(t, collector.IsRunning)

	assert.Greater(t, len(collector.FeedHashes), 0)
	collector.Stop()
	assert.False(t, collector.IsRunning)
}

func TestCollectorStream(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	time.Sleep(20 * time.Millisecond)

	collector := testItems.Collector
	assert.Greater(t, len(collector.FeedHashes), 0)
	assert.True(t, collector.IsRunning)

	headers := map[string]string{"X-API-Key": testItems.ApiKey}
	conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint(testItems.MockDal.URL+"/ws"), wss.WithRequestHeaders(headers))
	if err != nil {
		t.Fatalf("error creating websocket helper: %v", err)
	}

	err = conn.Dial(ctx)
	if err != nil {
		t.Fatalf("error dialing websocket: %v", err)
	}

	err = conn.Write(ctx, hub.Subscription{
		Method: "SUBSCRIBE",
		Params: []string{"submission@test-aggregate"},
	})
	if err != nil {
		t.Fatalf("error subscribing to websocket: %v", err)
	}

	sampleSubmissionData, err := generateSampleSubmissionData(
		testItems.TmpConfig.ID,
		int64(15),
		time.Now(),
		1,
		"test-aggregate",
	)
	if err != nil {
		t.Fatalf("error generating sample submission data: %v", err)
	}

	log.Debug().Msg("Publishing data")
	err = testPublishData(ctx, "test-aggregate", *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing data: %v", err)
	}
	log.Debug().Int32("configId", sampleSubmissionData.GlobalAggregate.ConfigID).Msg("Published data")

	ch := make(chan any)
	go conn.Read(ctx, ch)

	expected, err := testItems.Collector.IncomingDataToOutgoingData(ctx, sampleSubmissionData)
	if err != nil {
		t.Fatalf("error converting sample submission data to outgoing data: %v", err)
	}

	sample := <-ch
	result, err := wsfcommon.MessageToStruct[common.OutgoingSubmissionData](sample.(map[string]any))
	if err != nil {
		t.Fatalf("error converting sample to struct: %v", err)
	}
	assert.Equal(t, *expected, result)

	err = conn.Close()
	if err != nil {
		t.Fatalf("error closing websocket: %v", err)
	}
}
