package db

import (
	"context"
	"encoding/json"
	"reflect"
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

func TestRedisMGetEmptyKeys(t *testing.T) {
	ctx := context.Background()

	keys := []string{}
	values, err := MGet(ctx, keys)
	if err == nil {
		t.Errorf("Expected to have err")
	}
	if len(values) != 0 {
		t.Errorf("Expected 0 values, got %v", len(values))
	}
}

func TestRedisMGetInvaliedKeys(t *testing.T) {
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

	keys := []string{key1, key2, "testKey3"}
	values, err := MGet(ctx, keys)
	if err != nil {
		t.Errorf("Error getting keys: %v", err)
	}
	if len(values) != 3 {
		t.Errorf("Expected 2 values, got %v", len(values))
	}
	if values[0] != value1 {
		t.Errorf("Value did not match expected. Got %v, expected %v", values[0], value1)
	}
	if values[1] != value2 {
		t.Errorf("Value did not match expected. Got %v, expected %v", values[1], value2)
	}
	if values[2] != nil {
		t.Errorf("Value did not match expected. Got %v, expected %v", values[2], nil)
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
func TestMSetObject(t *testing.T) {
	ctx := context.Background()

	values := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	err := MSetObject(ctx, values)
	if err != nil {
		t.Errorf("Error setting objects: %v", err)
	}

	// Check if the values were set correctly
	for key, value := range values {
		gotValue, getValueErr := Get(ctx, key)
		if getValueErr != nil {
			t.Errorf("Error getting key: %v", getValueErr)
		}
		expectedValue, _ := json.Marshal(value)
		if gotValue != string(expectedValue) {
			t.Errorf("Value did not match expected. Got %v, expected %v", gotValue, string(expectedValue))
		}
	}

	// Clean up
	for key := range values {
		err = Del(ctx, key)
		if err != nil {
			t.Errorf("Error deleting key: %v", err)
		}
	}
}
func TestSetObject(t *testing.T) {
	ctx := context.Background()

	key := "testKey"
	value := "testValue"
	exp := 10 * time.Second

	err := SetObject(ctx, key, value, exp)
	if err != nil {
		t.Errorf("Error setting object: %v", err)
	}

	// Check if the value was set correctly
	gotValue, err := Get(ctx, key)
	if err != nil {
		t.Errorf("Error getting key: %v", err)
	}
	expectedValue, _ := json.Marshal(value)
	if gotValue != string(expectedValue) {
		t.Errorf("Value did not match expected. Got %v, expected %v", gotValue, string(expectedValue))
	}

	// Clean up
	err = Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}

func TestMGetObject(t *testing.T) {
	ctx := context.Background()

	type TestStruct struct {
		ID   int
		Name string
	}

	// Set up test data
	key1 := "testKey1"
	value1 := `{"ID": 1, "Name": "Test1"}`
	key2 := "testKey2"
	value2 := `{"ID": 2, "Name": "Test2"}`
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
	results, err := MGetObject[TestStruct](ctx, keys)
	if err != nil {
		t.Errorf("Error getting objects: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %v", len(results))
	}

	// Check the values of the retrieved objects
	expectedResult1 := TestStruct{ID: 1, Name: "Test1"}
	if !reflect.DeepEqual(results[0], expectedResult1) {
		t.Errorf("Result did not match expected. Got %+v, expected %+v", results[0], expectedResult1)
	}

	expectedResult2 := TestStruct{ID: 2, Name: "Test2"}
	if !reflect.DeepEqual(results[1], expectedResult2) {
		t.Errorf("Result did not match expected. Got %+v, expected %+v", results[1], expectedResult2)
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
func TestGetObject(t *testing.T) {
	ctx := context.Background()

	type TestStruct struct {
		ID   int
		Name string
	}

	// Set up test data
	key := "testKey"
	value := `{"ID": 1, "Name": "Test"}`
	exp := 10 * time.Second

	err := Set(ctx, key, value, exp)
	if err != nil {
		t.Errorf("Error setting key: %v", err)
	}

	// Test GetObject
	result, err := GetObject[TestStruct](ctx, key)
	if err != nil {
		t.Errorf("Error getting object: %v", err)
	}

	// Check the value of the retrieved object
	expectedResult := TestStruct{ID: 1, Name: "Test"}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Result did not match expected. Got %+v, expected %+v", result, expectedResult)
	}

	// Clean up
	err = Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}
