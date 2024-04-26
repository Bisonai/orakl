package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	_, err = RawPostRequest(testItems.app, "/api/v1/config/sync", nil)
	if err != nil {
		t.Fatalf("error syncing config: %v", err)
	}

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Greater(t, len(readResult), 1)
}

func TestConfigRead(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}
