package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
)

func TestGetRedisConnSingleton(t *testing.T) {
	ctx := context.Background()

	// Call GetRedisConn multiple times
	rdb1, err := db.GetRedisConn(ctx)
	if err != nil {
		t.Fatalf("GetRedisConn failed: %v", err)
	}
	defer db.CloseRedis()

	rdb2, err := db.GetRedisConn(ctx)
	if err != nil {
		t.Fatalf("GetRedisConn failed: %v", err)
	}

	// Check that the returned instances are the same
	if rdb1 != rdb2 {
		t.Errorf("GetRedisConn did not return the same instance")
	}
}
