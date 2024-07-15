//nolint:all
package test

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/utils/request"
	wsfcommon "bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"bisonai.com/orakl/node/pkg/wss"
	"github.com/stretchr/testify/assert"
)

func TestApiControllerRun(t *testing.T) {
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
	assert.True(t, testItems.Controller.Collector.IsRunning)
}

func TestApiGetLatestAll(t *testing.T) {
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

	err = testPublishData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing sample submission data: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	result, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint("http://localhost:8090/api/v1/dal/latest-data-feeds/all"), request.WithHeaders(map[string]string{"X-API-Key": testItems.ApiKey}))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}
	expected, err := testItems.Controller.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error converting sample submission data to outgoing data: %v", err)
	}

	assert.Greater(t, len(result), 0)
	if len(result) > 0 {
		assert.Equal(t, *expected, result[0])
	}
}

func TestShouldFailWithoutApiKey(t *testing.T) {
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
	resp, err := request.RequestRaw(request.WithEndpoint("http://localhost:8090/api/v1"))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}

	assert.Equal(t, 200, resp.StatusCode)

	result, err := request.RequestRaw(request.WithEndpoint("http://localhost:8090/api/v1/dal/latest-data-feeds/test-aggregate"))

	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}

	assert.Equal(t, 401, result.StatusCode)
}

func TestApiGetLatest(t *testing.T) {
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

	err = testPublishData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing sample submission data: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	result, err := request.Request[common.OutgoingSubmissionData](request.WithEndpoint("http://localhost:8090/api/v1/dal/latest-data-feeds/test-aggregate"), request.WithHeaders(map[string]string{"X-API-Key": testItems.ApiKey}))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}
	expected, err := testItems.Controller.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error converting sample submission data to outgoing data: %v", err)
	}
	assert.Equal(t, *expected, result)
}

func TestApiWebsocket(t *testing.T) {
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

	headers := map[string]string{"X-API-Key": testItems.ApiKey}

	go testItems.App.Listen(":8090")

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

	err = testPublishData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing sample submission data: %v", err)
	}

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
