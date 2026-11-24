package db

import (
	"fmt"
	"os"
	"testing"

	"bisonai.com/miko/node/pkg/testutil/testenv"
)

// externalRedis reports whether the tests talk to a real Redis (REDIS_HOST
// set by the caller) rather than the in-process miniredis fallback.
var externalRedis bool

func TestMain(m *testing.M) {
	externalRedis = os.Getenv("REDIS_HOST") != ""
	stopRedis, err := testenv.StartRedis()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	ClosePool()
	CloseRedis()
	stopRedis()
	os.Exit(code)
}
