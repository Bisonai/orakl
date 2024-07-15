//nolint:all

package test

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/common"
	wsfcommon "bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
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

	collector := testItems.Controller.Collector
	assert.True(t, collector.IsRunning)

	assert.Greater(t, len(collector.Symbols), 0)
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
	go testItems.App.Listen(":8090")

	time.Sleep(20 * time.Millisecond)

	collector := testItems.Controller.Collector
	assert.Greater(t, len(collector.Symbols), 0)
	assert.True(t, collector.IsRunning)

	headers := map[string]string{"X-API-Key": testItems.ApiKey}
	conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint("ws://localhost:8090/api/v1/dal/ws"), wss.WithRequestHeaders(headers))
	if err != nil {
		t.Fatalf("error creating websocket helper: %v", err)
	}

	err = conn.Dial(ctx)
	if err != nil {
		t.Fatalf("error dialing websocket: %v", err)
	}

	err = conn.Write(ctx, api.Subscription{
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

	log.Info().Msg("Publishing data")
	err = testPublishData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing data: %v", err)
	}
	log.Info().Int32("configId", sampleSubmissionData.GlobalAggregate.ConfigID).Msg("Published data")

	ch := make(chan any)
	go conn.Read(ctx, ch)

	expected, err := testItems.Controller.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
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
