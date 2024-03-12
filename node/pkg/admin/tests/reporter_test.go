//nolint:all
package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestReporterActivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.REPORTER)
	waitForMessage(t, channel, bus.ADMIN, bus.REPORTER, bus.ACTIVATE_REPORTER)

	result, err := RawPostRequest(testItems.app, "/api/v1/reporter/activate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	assert.Equal(t, string(result), "reporter activated: true")
}

func TestReporterDeactivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.REPORTER)
	waitForMessage(t, channel, bus.ADMIN, bus.REPORTER, bus.DEACTIVATE_REPORTER)

	result, err := RawPostRequest(testItems.app, "/api/v1/reporter/deactivate", nil)
	if err != nil {
		t.Fatalf("error deactivating reporter: %v", err)
	}

	assert.Equal(t, string(result), "reporter deactivated: true")
}

func TestReporterRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.REPORTER)
	waitForMessage(t, channel, bus.ADMIN, bus.REPORTER, bus.REFRESH_REPORTER)

	result, err := RawPostRequest(testItems.app, "/api/v1/reporter/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing reporter: %v", err)
	}

	assert.Equal(t, string(result), "reporter refreshed: true")
}
