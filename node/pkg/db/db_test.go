package db

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// setup
	code := m.Run()
	ClosePool()
	CloseRedis()
	// teardown
	os.Exit(code)
}
