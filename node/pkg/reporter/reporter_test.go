//nolint:all
package reporter

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/tests"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporter.isRunning, true)
}

func TestStopReporter(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatal("error running reporter")
	}

	err = testItems.app.stopReporter()
	if err != nil {
		t.Fatal("error stopping reporter")
	}

	assert.Equal(t, testItems.app.Reporter.isRunning, false)
}

func TestStopReporterByAdmin(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	testItems.app.subscribe(ctx)

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatal("error running reporter")
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/deactivate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporter.isRunning, false)
}

func TestStartReporterByAdmin(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	testItems.app.subscribe(ctx)
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/activate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporter.isRunning, true)
}

func TestRestartReporterByAdmin(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	testItems.app.subscribe(ctx)
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/activate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporter.isRunning, true)
}
