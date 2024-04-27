package tests

import (
	"context"
	"strconv"
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
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

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

func TestConfigInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	insertResult, err := PostRequest[config.ConfigModel](testItems.app, "/api/v1/config", config.ConfigModel{
		Name:              "test",
		Address:           "test",
		FetchInterval:     nil,
		AggregateInterval: nil,
		SubmitInterval:    nil,
	})
	if err != nil {
		t.Fatalf("error inserting config: %v", err)
	}
	assert.NotEqual(t, 0, insertResult.Id)

}

func TestConfigRead(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestConfigReadById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	readResult, err := GetRequest[config.ConfigModel](testItems.app, "/api/v1/config/"+strconv.Itoa(int(testItems.tmpData.config.Id)), nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Equal(t, testItems.tmpData.config.Id, readResult.Id)
}

func TestConfigDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	deleted, err := DeleteRequest[config.ConfigModel](testItems.app, "/api/v1/config/"+strconv.Itoa(int(testItems.tmpData.config.Id)), nil)
	if err != nil {
		t.Fatalf("error deleting config: %v", err)
	}
	assert.Equal(t, testItems.tmpData.config.Id, deleted.Id)

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}

	assert.Equal(t, 0, len(readResult))

}
