//nolint:all

package test

import (
	"context"
	"testing"
	"time"

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

	collector := testItems.Collector
	collector.Start(ctx)
	assert.NotEqual(t, nil, collector.Ctx)
	assert.Greater(t, len(collector.Symbols), 0)
	assert.Greater(t, len(collector.Symbols), 0)

	collector.Stop()
	assert.Equal(t, nil, collector.Ctx)
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

	collector := testItems.Collector
	collector.Start(ctx)
	time.Sleep(10 * time.Millisecond)

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

	testPublishData(ctx, *sampleSubmissionData)

	time.Sleep(10 * time.Millisecond)

	expected, err := collector.IncomingDataToOutgoingData(ctx, *sampleSubmissionData)
	if err != nil {
		t.Fatalf("error converting incoming data to outgoing data: %v", err)
	}

	select {
	case sample := <-collector.OutgoingStream[testItems.TmpConfig.ID]:
		assert.NotEqual(t, nil, sample)
		assert.Equal(t, *expected, sample)
	default:
		t.Fatalf("no data received")
	}
}
