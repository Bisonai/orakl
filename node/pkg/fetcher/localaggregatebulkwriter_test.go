package fetcher

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestLocalAggregateBulkWriter(t *testing.T) {
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

	go testItems.app.Run(ctx)

	time.Sleep(DefaultLocalAggregateInterval * 20)

	pgsqlData, pgsqlErr := db.QueryRows[LocalAggregate](ctx, "SELECT * FROM local_aggregates", nil)
	if pgsqlErr != nil {
		t.Fatalf("error getting local aggregate from pgsql: %v", pgsqlErr)
	}
	assert.Greater(t, len(pgsqlData), 0)

}
