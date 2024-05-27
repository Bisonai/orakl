package db

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
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

	var pairs []interface{}
	for key, value := range values {
		pairs = append(pairs, key, value)
	}
	return rdbConn.MSet(ctx, pairs...).Err()
}

func MSetObject(ctx context.Context, values map[string]any) error {
	stringMap := make(map[string]string)
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling object")
			return err
		}
		stringMap[key] = string(data)
	}
	return MSet(ctx, stringMap)
}

func Set(ctx context.Context, key string, value string, exp time.Duration) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return err
	}
	return setRedis(ctx, rdbConn, key, value, exp)
}

func SetObject(ctx context.Context, key string, value any, exp time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling object")
		return err
	}
	return Set(ctx, key, string(data), exp)
}

func MGet(ctx context.Context, keys []string) ([]any, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return nil, err
	}
	return rdbConn.MGet(ctx, keys...).Result()
}

func MGetObject[T any](ctx context.Context, keys []string) ([]T, error) {
	results := []T{}

	data, err := MGet(ctx, keys)
	if err != nil {
		log.Error().Strs("keys", keys).Err(err).Msg("Error getting objects from MGetObject")
		return results, err
	}

	for _, d := range data {
		if d == nil {
			log.Warn().Msg("Nil value in MGetObject")
			continue
		}
		var t T
		err = json.Unmarshal([]byte(d.(string)), &t)
		if err != nil {
			log.Warn().Err(err).Msg("Error unmarshalling object")
			continue
		}
		results = append(results, t)
	}
	return results, nil
}

func Get(ctx context.Context, key string) (string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return "", err
	}
	return getRedis(ctx, rdbConn, key)
}

func GetObject[T any](ctx context.Context, key string) (T, error) {
	var t T
	data, err := Get(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Error getting object")
		return t, err
	}
	err = json.Unmarshal([]byte(data), &t)
	return t, err
}

func Del(ctx context.Context, key string) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return err
	}
	return rdbConn.Del(ctx, key).Err()
}

func LRange(ctx context.Context, key string, start int64, end int64) ([]string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return nil, err
	}
	return rdbConn.LRange(ctx, key, start, end).Result()
}

func LPush(ctx context.Context, key string, values ...any) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return err
	}
	return rdbConn.LPush(ctx, key, values...).Err()
}

func LPushObject(ctx context.Context, key string, values []any) error {
	stringValues := make([]interface{}, len(values))
	for i, v := range values {
		data, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling object")
			return err
		}
		stringValues[i] = string(data)
	}
	return LPush(ctx, key, stringValues...)
}

func PopAll(ctx context.Context, key string) ([]string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return nil, err
	}

	pipe := rdbConn.TxPipeline()
	lrange := pipe.LRange(ctx, key, 0, -1)
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error executing pipeline")
		return nil, err
	}

	return lrange.Val(), nil
}

func PopAllObject[T any](ctx context.Context, key string) ([]T, error) {
	data, err := PopAll(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Error popping all objects")
		return nil, err
	}

	results := []T{}
	for _, d := range data {
		var t T
		err = json.Unmarshal([]byte(d), &t)
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling object")
			return nil, err
		}
		results = append(results, t)
	}
	return results, nil
}

func LRange(ctx context.Context, key string, start int64, end int64) ([]string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
	}
	return rdbConn.LRange(ctx, key, start, end).Result()
}

func LPush(ctx context.Context, key string, values ...any) error {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return err
	}
	return rdbConn.LPush(ctx, key, values...).Err()
}

func LPushObject(ctx context.Context, key string, values []any) error {
	stringValues := make([]interface{}, len(values))
	for i, v := range values {
		data, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling object")
			return err
		}
		stringValues[i] = string(data)
	}
	return LPush(ctx, key, stringValues...)
}

func PopAll(ctx context.Context, key string) ([]string, error) {
	rdbConn, err := GetRedisConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting redis connection")
		return nil, err
	}

	pipe := rdbConn.TxPipeline()
	lrange := pipe.LRange(ctx, key, 0, -1)
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error executing pipeline")
		return nil, err
	}

	return lrange.Val(), nil
}

func PopAllObject[T any](ctx context.Context, key string) ([]T, error) {
	data, err := PopAll(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Error popping all objects")
		return nil, err
	}

	results := []T{}
	for _, d := range data {
		var t T
		err = json.Unmarshal([]byte(d), &t)
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling object")
			return nil, err
		}
		results = append(results, t)
	}
	return results, nil
}

func connectToRedis(ctx context.Context, connectionInfo RedisConnectionInfo) (*redis.Conn, error) {
	rdbConn := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port}).Conn()
	_, rdbErr := rdbConn.Ping(ctx).Result()
	if rdbErr != nil {
		log.Error().Err(rdbErr).Msg("Error connecting to redis")
		return nil, rdbErr
	}
	return rdbConn, nil
}

func loadRedisConnectionString() (RedisConnectionInfo, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		log.Error().Msg("REDIS_HOST not set")
		return RedisConnectionInfo{}, errorSentinel.ErrRdbHostNotFound
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		log.Error().Msg("REDIS_PORT not set")
		return RedisConnectionInfo{}, errorSentinel.ErrRdbPortNotFound
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
