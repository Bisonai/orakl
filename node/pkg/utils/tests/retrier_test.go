package tests

import (
	"errors"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/utils/retrier"
)

func TestRetry_MaxAttemptsReached(t *testing.T) {
	// Define a mock job function that always returns an error
	mockJob := func() error {
		return errors.New("mock error")
	}

	// Define the maximum number of attempts, initial timeout, and maximum timeout
	maxAttempts := 3
	initialTimeout := time.Second
	maxTimeout := 5 * time.Second

	// Call the Retry function with the mock job and parameters
	err := retrier.Retry(mockJob, maxAttempts, initialTimeout, maxTimeout)

	// Check if the Retry function returned the expected error

	if err == nil {
		t.Error("Expected an error, but got nil")
	}
}

func TestRetry_SuccessfulJob(t *testing.T) {
	// Define a mock job function that always returns nil (no error)
	mockJob := func() error {
		return nil
	}

	// Define the maximum number of attempts, initial timeout, and maximum timeout
	maxAttempts := 3
	initialTimeout := time.Second
	maxTimeout := 5 * time.Second

	// Call the Retry function with the mock job and parameters
	err := retrier.Retry(mockJob, maxAttempts, initialTimeout, maxTimeout)

	// Check if the Retry function returned nil (no error)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
