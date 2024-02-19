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
	return getRedisConn(ctx, &initRdbOnce)
}

func getRedisConn(ctx context.Context, once *sync.Once) (*redis.Conn, error) {
	var err error
	once.Do(func() {
		rdb, err = connectRdb(ctx)
	})
	return rdb, err

}

func connectRdb(ctx context.Context) (*redis.Conn, error) {
	connectionInfo, err := loadRedisConnectionString()
	if err != nil {
		return nil, err
	}
	return connectToRedis(ctx, connectionInfo)

}

func Set(ctx context.Context, key string, value string, exp time.Duration) error {
	rdb, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	return setRedis(ctx, rdb, key, value, exp)
}

func Get(ctx context.Context, key string) (string, error) {
	rdb, err := GetRedisConn(ctx)
	if err != nil {
		return "", err
	}
	return getRedis(ctx, rdb, key)
}

func Del(ctx context.Context, key string) error {
	rdb, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	return rdb.Del(ctx, key).Err()
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

func setRedis(ctx context.Context, rdb *redis.Conn, key string, value string, exp time.Duration) error {
	return rdb.Set(ctx, key, value, exp).Err()
}

func getRedis(ctx context.Context, rdb *redis.Conn, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
}

func CloseRedis() {
	if rdb != nil {
		rdb.Close()
		rdb = nil
	}
}
