package db

import (
	"context"
	"testing"
	"time"
)

func TestGetRedisConnSingleton(t *testing.T) {
	ctx := context.Background()
	// Call GetRedisConn multiple times
	rdb1, err := GetRedisConn(ctx)
	if err != nil {
		t.Fatalf("GetRedisConn failed: %v", err)
	}

	rdb2, err := GetRedisConn(ctx)
	if err != nil {
		t.Fatalf("GetRedisConn failed: %v", err)
	}

	// Check that the returned instances are the same
	if rdb1 != rdb2 {
		t.Errorf("GetRedisConn did not return the same instance")
	}

}

func TestRedisGetSet(t *testing.T) {
	ctx := context.Background()

	key := "testKey"
	value := "testValue"
	exp := 10 * time.Second

	// Test Set
	err := Set(ctx, key, value, exp)
	if err != nil {
		t.Errorf("Error setting key: %v", err)
	}

	// Test Get
	gotValue, err := Get(ctx, key)
	if err != nil {
		t.Errorf("Error getting key: %v", err)
	}
	if gotValue != value {
		t.Errorf("Value did not match expected. Got %v, expected %v", gotValue, value)
	}

	// Clean up
	err = Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}

func TestRedisMGet(t *testing.T) {
	ctx := context.Background()

	key1 := "testKey1"
	value1 := "testValue1"
	key2 := "testKey2"
	value2 := "testValue2"
	exp := 10 * time.Second

	err := Set(ctx, key1, value1, exp)
	if err != nil {
		t.Errorf("Error setting key: %v", err)
	}
	err = Set(ctx, key2, value2, exp)
	if err != nil {
		t.Errorf("Error setting key: %v", err)
	}

	keys := []string{key1, key2}
	values, err := MGet(ctx, keys)
	if err != nil {
		t.Errorf("Error getting keys: %v", err)
	}
	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %v", len(values))
	}
	if values[0] != value1 {
		t.Errorf("Value did not match expected. Got %v, expected %v", values[0], value1)
	}
	if values[1] != value2 {
		t.Errorf("Value did not match expected. Got %v, expected %v", values[1], value2)
	}

	// Clean up
	err = Del(ctx, key1)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
	err = Del(ctx, key2)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}
