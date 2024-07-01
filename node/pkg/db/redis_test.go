package db

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetRedisConnSingleton(t *testing.T) {
	ctx := context.Background()
	// Call GetRedisConn multiple times
	rdb1, err := GetRedisClient(ctx)
	if err != nil {
		t.Fatalf("GetRedisConn failed: %v", err)
	}

	rdb2, err := GetRedisClient(ctx)
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
		expectedValue, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Errorf("Error marshalling value: %v", marshalErr)
			continue
		}
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
	expectedValue, err := json.Marshal(value)
	if err != nil {
		t.Errorf("Error marshalling value: %v", err)
	}
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

func TestLPush(t *testing.T) {
	ctx := context.Background()

	key := "testKey"
	values := []string{"value1", "value2", "value3"}

	err := LPush(ctx, key, values)
	if err != nil {
		t.Errorf("Error pushing values to list: %v", err)
	}

	// Check if the values were pushed correctly
	result, err := LRange(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("Error getting list: %v", err)
	}

	expectedResult := []string{"value3", "value2", "value1"}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Result did not match expected. Got %+v, expected %+v", result, expectedResult)
	}

	// Clean up
	err = Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}

func TestLPushObject(t *testing.T) {
	ctx := context.Background()

	type TestStruct struct {
		ID   int
		Name string
	}

	key := "testKey"
	values := []TestStruct{
		{ID: 1, Name: "Test1"},
		{ID: 2, Name: "Test2"},
		{ID: 3, Name: "Test3"},
	}

	err := LPushObject(ctx, key, values)
	if err != nil {
		t.Errorf("Error pushing objects to list: %v", err)
	}

	// Check if the objects were pushed correctly
	result, err := LRange(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("Error getting list: %v", err)
	}

	expectedResult := []string{}
	for _, v := range values {
		data, marshallErr := json.Marshal(v)
		if marshallErr != nil {
			t.Errorf("Error marshalling object: %v", marshallErr)
		}
		expectedResult = append(expectedResult, string(data))
	}

	sort.Strings(result)
	sort.Strings(expectedResult)

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Result did not match expected. Got %+v, expected %+v", result, expectedResult)
	}

	// Clean up
	err = Del(ctx, key)
	if err != nil {
		t.Errorf("Error deleting key: %v", err)
	}
}

func TestPopAll(t *testing.T) {
	ctx := context.Background()

	key := "testKey"
	values := []string{"value1", "value2", "value3"}

	// Push values to the list
	err := LPush(ctx, key, values)
	if err != nil {
		t.Fatalf("Error pushing values to list: %v", err)
	}

	// Call PopAll
	poppedValues, err := PopAll(ctx, key)
	if err != nil {
		t.Fatalf("Error popping all values: %v", err)
	}

	// Check if the popped values match the expected values
	expectedValues := []string{"value3", "value2", "value1"}
	assert.ObjectsAreEqualValues(expectedValues, poppedValues)

	// Check if the list is empty after popping
	result, err := Get(ctx, key)
	if err == nil || !strings.Contains(err.Error(), "redis: nil") {
		t.Fatalf("Expected to have err")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty list, got %+v", result)
	}
}

func TestPopAllObject(t *testing.T) {
	ctx := context.Background()

	type TestStruct struct {
		ID   int
		Name string
	}

	key := "testKey"
	values := []TestStruct{
		{ID: 1, Name: "Test1"},
		{ID: 2, Name: "Test2"},
		{ID: 3, Name: "Test3"},
	}

	// Push objects to the list
	err := LPushObject(ctx, key, values)
	if err != nil {
		t.Fatalf("Error pushing objects to list: %v", err)
	}

	// Call PopAllObject
	poppedValues, err := PopAllObject[TestStruct](ctx, key)
	if err != nil {
		t.Fatalf("Error popping all objects: %v", err)
	}

	// Check if the popped objects match the expected objects
	expectedValues := []TestStruct{
		{ID: 3, Name: "Test3"},
		{ID: 2, Name: "Test2"},
		{ID: 1, Name: "Test1"},
	}
	if !reflect.DeepEqual(poppedValues, expectedValues) {
		t.Errorf("Popped objects did not match expected. Got %+v, expected %+v", poppedValues, expectedValues)
	}

	// Check if the list is empty after popping
	result, err := Get(ctx, key)
	if err == nil || !strings.Contains(err.Error(), "redis: nil") {
		t.Fatalf("Expected to have err")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty list, got %+v", result)
	}
}
func TestMSetObjectWithExp(t *testing.T) {
	ctx := context.Background()

	values := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	exp := 1 * time.Second

	err := MSetObjectWithExp(ctx, values, exp)
	if err != nil {
		t.Errorf("Error setting objects with expiration: %v", err)
	}

	// Check if the values were set correctly
	for key, value := range values {
		gotValue, getValueErr := Get(ctx, key)
		if getValueErr != nil {
			t.Errorf("Error getting key: %v", getValueErr)
		}
		expectedValue, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Errorf("Error marshalling value: %v", marshalErr)
			continue
		}
		if gotValue != string(expectedValue) {
			t.Errorf("Value did not match expected. Got %v, expected %v", gotValue, string(expectedValue))
		}
	}

	time.Sleep(1001 * time.Millisecond)

	// Check if the values were expired
	for key := range values {
		gotValue, getValueErr := Get(ctx, key)
		if getValueErr == nil || !strings.Contains(getValueErr.Error(), "redis: nil") {
			t.Errorf("Expected to have err")
		}
		if gotValue != "" {
			t.Errorf("Expected empty value, got %v", gotValue)
		}
	}
}
