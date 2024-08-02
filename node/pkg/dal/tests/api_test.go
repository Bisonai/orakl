//nolint:all
package test

import (
	"context"
	"sync"
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
	assert.True(t, testItems.Collector.IsRunning)
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
	result, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint("http://localhost:8090/latest-data-feeds/all"), request.WithHeaders(map[string]string{"X-API-Key": testItems.ApiKey}))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}
	expected, err := testItems.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
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
	resp, err := request.RequestRaw(request.WithEndpoint("http://localhost:8090/"))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}

	assert.Equal(t, 200, resp.StatusCode)

	result, err := request.RequestRaw(request.WithEndpoint("http://localhost:8090/latest-data-feeds/test-aggregate"))

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

	result, err := request.Request[[]common.OutgoingSubmissionData](request.WithEndpoint("http://localhost:8090/latest-data-feeds/test-aggregate"), request.WithHeaders(map[string]string{"X-API-Key": testItems.ApiKey}))
	if err != nil {
		t.Fatalf("error getting latest data: %v", err)
	}
	expected, err := testItems.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error converting sample submission data to outgoing data: %v", err)
	}
	assert.Equal(t, *expected, result[0])
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

	t.Run("test subscription", func(t *testing.T) {

		conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint("ws://localhost:8090/ws"), wss.WithRequestHeaders(headers))
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

		expected, err := testItems.Collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
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
	})

	t.Run("test fail for 10+ dial", func(t *testing.T) {
		conns := []*wss.WebsocketHelper{}
		defer func() {
			for _, conn := range conns {
				err := conn.Close()
				if err != nil {
					t.Logf("error closing websocket: %v", err)
				}
			}
		}()
		for i := 0; i < 11; i++ {
			conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint("ws://localhost:8090/ws"), wss.WithRequestHeaders(headers))
			if err != nil {
				t.Fatalf("error creating websocket helper: %v", err)
			}

			err = conn.Dial(ctx)
			if err != nil {
				t.Fatalf("error dialing websocket: %v", err)
			}

			conns = append(conns, conn)
		}
		time.Sleep(1 * time.Second)

		assert.Error(t, conns[0].IsAlive(ctx), "expected to fail due to too many connections")

	})

	t.Run("test multiconnection and multi subscriptions", func(t *testing.T) {
		var wg sync.WaitGroup
		headers := map[string]string{"X-API-Key": testItems.ApiKey}
		connCount := 10
		subscriptions := []string{"submission@test-aggregate"}

		// Create a channel to collect all results
		resultsChan := make(chan common.OutgoingSubmissionData, connCount*len(subscriptions))

		for i := 0; i < connCount; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint("ws://localhost:8090/ws"), wss.WithRequestHeaders(headers))
				if err != nil {
					t.Errorf("error creating websocket helper for client %d: %v", clientID, err)
					return
				}

				err = conn.Dial(ctx)
				if err != nil {
					t.Errorf("error dialing websocket for client %d: %v", clientID, err)
					return
				}

				defer func() {
					err = conn.Close()
					if err != nil {
						t.Errorf("error closing websocket for client %d: %v", clientID, err)
					}
				}()

				err = conn.Write(ctx, api.Subscription{
					Method: "SUBSCRIBE",
					Params: subscriptions,
				})
				if err != nil {
					t.Errorf("error subscribing for client %d: %v", clientID, err)
					return
				}

				// Receive messages
				ch := make(chan any)
				go conn.Read(ctx, ch)

				// Read messages from the channel and store the results
				for j := 0; j < len(subscriptions); j++ {
					select {
					case sample := <-ch:
						result, err := wsfcommon.MessageToStruct[common.OutgoingSubmissionData](sample.(map[string]any))
						if err != nil {
							t.Errorf("error converting sample to struct for client %d: %v", clientID, err)
							return
						}
						resultsChan <- result
					case <-time.After(10 * time.Second): // Timeout if no message is received
						t.Errorf("timeout waiting for message for client %d", clientID)
						return
					}
				}
			}(i)
		}

		// Simulate data publication
		expectedData, err := generateSampleSubmissionData(
			testItems.TmpConfig.ID,
			int64(15),
			time.Now(),
			1,
			"test-aggregate",
		)
		if err != nil {
			t.Fatalf("error generating expected data: %v", err)
		}
		expected, err := testItems.Collector.IncomingDataToOutgoingData(ctx, *expectedData)
		if err != nil {
			t.Fatalf("error converting sample submission data to outgoing data: %v", err)
		}

		// Publish data
		err = testPublishData(ctx, *expectedData)
		if err != nil {
			t.Fatalf("error publishing sample submission data: %v", err)
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(resultsChan)

		// Verify results
		for result := range resultsChan {
			if result.Symbol == expected.Symbol {
				assert.Equal(t, *expected, result)
			} else {
				t.Errorf("unexpected data received: %v", result)
			}
		}
	})
}
