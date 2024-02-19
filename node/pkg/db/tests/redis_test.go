package tests

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
)

func TestRedisSetAndGet(t *testing.T) {
	ctx := context.Background()
	key := "testKey"
	value := "testValue"
	exp := 10 * time.Second

	// Test Set
	err := db.Set(ctx, key, value, exp)
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

	defer db.CloseRedis()
}
