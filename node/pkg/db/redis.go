package db

import (
	"context"
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

func GetRedisConn() (*redis.Conn, error) {
	var err error
	initRdbOnce.Do(func() {
		connectionInfo := loadRedisConnectionString()
		rdb, err = connectToRedis(connectionInfo)
	})
	return rdb, err
}

func Set(key string, value string, exp time.Duration) error {
	rdb, err := GetRedisConn()
	if err != nil {
		return err
	}
	return setRedis(rdb, key, value, exp)
}

func Get(key string) (string, error) {
	rdb, err := GetRedisConn()
	if err != nil {
		return "", err
	}
	return getRedis(rdb, key)
}

func connectToRedis(connectionInfo RedisConnectionInfo) (*redis.Conn, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port}).Conn()
	_, rdbErr := rdb.Ping(context.Background()).Result()
	if rdbErr != nil {
		return nil, rdbErr
	}
	return rdb, nil
}

func loadRedisConnectionString() RedisConnectionInfo {
	return RedisConnectionInfo{
		Host: os.Getenv("REDIS_HOST"),
		Port: os.Getenv("REDIS_PORT"),
	}
}

func setRedis(rdb *redis.Conn, key string, value string, exp time.Duration) error {
	return rdb.Set(context.Background(), key, value, exp).Err()
}

func getRedis(rdb *redis.Conn, key string) (string, error) {
	return rdb.Get(context.Background(), key).Result()
}
