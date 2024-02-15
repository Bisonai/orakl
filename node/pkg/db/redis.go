package db

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// singleton pattern
// make sure env is loaded from main before calling this

type RedisConnectionInfo struct {
	Host string
	Port string
}

var (
	initRdbOnce sync.Once
	rdb         *redis.Conn
)

func GetRedisConn(ctx context.Context) (*redis.Conn, error) {
	var err error

	initRdbOnce.Do(func() {
		connectionInfo, initErr := loadRedisConnectionString()
		if initErr != nil {
			err = initErr
			return
		}
		rdb, err = connectToRedis(ctx, connectionInfo)
	})
	return rdb, err
}

func Set(ctx context.Context, key string, value string, exp time.Duration) error {
	rdb, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	return setRedis(rdb, key, value, exp)
}

func Get(ctx context.Context, key string) (string, error) {
	rdb, err := GetRedisConn(ctx)
	if err != nil {
		return "", err
	}
	return getRedis(rdb, key)
}

func connectToRedis(ctx context.Context, connectionInfo RedisConnectionInfo) (*redis.Conn, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port}).Conn()
	_, rdbErr := rdb.Ping(ctx).Result()
	if rdbErr != nil {
		return nil, rdbErr
	}
	return rdb, nil
}

func loadRedisConnectionString() (RedisConnectionInfo, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return RedisConnectionInfo{}, errors.New("REDIS_HOST not set")
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		return RedisConnectionInfo{}, errors.New("REDIS_PORT not set")
	}

	return RedisConnectionInfo{Host: host, Port: port}, nil
}

func setRedis(rdb *redis.Conn, key string, value string, exp time.Duration) error {
	return rdb.Set(context.Background(), key, value, exp).Err()
}

func getRedis(rdb *redis.Conn, key string) (string, error) {
	return rdb.Get(context.Background(), key).Result()
}
