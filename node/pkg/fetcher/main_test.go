package fetcher

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
)

func TestMain(m *testing.M) {
	// setup

	db.GetPool(context.Background())
	db.GetRedisConn(context.Background())
	code := m.Run()

	db.ClosePool()
	db.CloseRedis()

	// teardown
	os.Exit(code)
}
