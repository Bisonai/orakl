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
	rdbErr      error
)

func GetRedisConn(ctx context.Context) (*redis.Conn, error) {
	return getRedisConn(ctx, &initRdbOnce)
}

func getRedisConn(ctx context.Context, once *sync.Once) (*redis.Conn, error) {

	once.Do(func() {
		connectionInfo, err := loadRedisConnectionString()
		if err != nil {
			rdbErr = err
			return
		}

		rdb, rdbErr = connectToRedis(ctx, connectionInfo)
	})
	return rdb, rdbErr
}

func MSet(ctx context.Context, values map[string]string) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	rdb = rdbConn
	var pairs []interface{}
	for key, value := range values {
		pairs = append(pairs, key, value)
	}
	return rdb.MSet(ctx, pairs...).Err()
}

func Set(ctx context.Context, key string, value string, exp time.Duration) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	rdb = rdbConn
	return setRedis(ctx, rdb, key, value, exp)
}

func MGet(ctx context.Context, keys []string) ([]any, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		return nil, err
	}
	rdb = rdbConn
	return rdb.MGet(ctx, keys...).Result()
}

func Get(ctx context.Context, key string) (string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		return "", err
	}
	rdb = rdbConn
	return getRedis(ctx, rdb, key)
}

func Del(ctx context.Context, key string) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		return err
	}
	rdb = rdbConn
	return rdb.Del(ctx, key).Err()
}

func connectToRedis(ctx context.Context, connectionInfo RedisConnectionInfo) (*redis.Conn, error) {
	rdbConn := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port}).Conn()
	_, rdbErr := rdbConn.Ping(ctx).Result()
	if rdbErr != nil {
		return nil, rdbErr
	}
	return rdbConn, nil
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
