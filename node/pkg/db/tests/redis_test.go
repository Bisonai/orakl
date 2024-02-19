package tests

import (
	"context"
	"testing"
	"time"

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

	key := "testKey"
	value := "testValue"
	exp := 10 * time.Second

	// Test Set
	err = db.Set(ctx, key, value, exp)
	if err != nil {
		t.Errorf("Error setting key: %v", err)
	}

	// Test Get
	gotValue, err := db.Get(ctx, key)
	if err != nil {
		t.Errorf("Error getting key: %v", err)
	}
	if gotValue != value {
		t.Errorf("Value did not match expected. Got %v, expected %v", gotValue, value)
	}

	// Clean up
	err = db.Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}
