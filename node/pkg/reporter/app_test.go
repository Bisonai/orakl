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
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporters[0].isRunning, true)
}

func TestStopReporter(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatal("error running reporter")
	}

	err = testItems.app.stopReporters()
	if err != nil {
		t.Fatal("error stopping reporter")
	}

	assert.Equal(t, testItems.app.Reporters[0].isRunning, false)
}

func TestStopReporterByAdmin(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	testItems.app.subscribe(ctx)

	err = testItems.app.Run(ctx)
	if err != nil {
		t.Fatal("error running reporter")
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/deactivate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporters[0].isRunning, false)
}

func TestStartReporterByAdmin(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	testItems.app.subscribe(ctx)
	err = testItems.app.setReporters(ctx)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/activate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporters[0].isRunning, true)
}

func TestRestartReporterByAdmin(t *testing.T) {
	// TODO: add checking for address mapping changes

	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	testItems.app.subscribe(ctx)
	err = testItems.app.setReporters(ctx)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/activate", nil)
	if err != nil {
		t.Fatalf("error activating reporter: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/reporter/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing reporter: %v", err)
	}

	assert.Equal(t, testItems.app.Reporters[0].isRunning, true)
}
