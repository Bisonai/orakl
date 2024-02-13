package db

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConnectionInfo struct {
	Host string
	Port string
}

type RedisHelper struct {
	*redis.Conn
}

func NewRedisHelper() (*RedisHelper, error) {
	connectionInfo := LoadRedisConnectionString()
	rdb, err := ConnectToRedis(connectionInfo)
	if err != nil {
		return nil, err
	}
	return &RedisHelper{rdb}, nil
}

func (r *RedisHelper) Set(key string, value string, exp time.Duration) error {
	return SetRedis(r.Conn, key, value, exp)
}

func (r *RedisHelper) Get(key string) (string, error) {
	return GetRedis(r.Conn, key)
}

func ConnectToRedis(connectionInfo RedisConnectionInfo) (*redis.Conn, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port}).Conn()
	_, rdbErr := rdb.Ping(context.Background()).Result()
	if rdbErr != nil {
		return nil, rdbErr
	}
	return rdb, nil
}

func LoadRedisConnectionString() RedisConnectionInfo {
	return RedisConnectionInfo{
		Host: os.Getenv("REDIS_HOST"),
		Port: os.Getenv("REDIS_PORT"),
	}
}

func SetRedis(rdb *redis.Conn, key string, value string, exp time.Duration) error {
	return rdb.Set(context.Background(), key, value, exp).Err()
}

func GetRedis(rdb *redis.Conn, key string) (string, error) {
	return rdb.Get(context.Background(), key).Result()
}
