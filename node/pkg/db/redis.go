package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/retrier"
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
	rdbMutex sync.Mutex
	rdb      *redis.Client
)

func GetRedisConn(ctx context.Context) (*redis.Client, error) {
	rdbMutex.Lock()
	defer rdbMutex.Unlock()

	if rdb != nil {
		return rdb, nil
	}

	err := reconnectRedis(ctx)
	return rdb, err
}

func reconnectRedis(ctx context.Context) error {
	connectionInfo, err := loadRedisConnectionString()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load Redis connection string")
		return err
	}

	rdb, err = connectToRedis(ctx, connectionInfo)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis")
		return err
	}
	return nil
}

func executeWithRetry(ctx context.Context, operation func(*redis.Client) error) error {
	retryOperation := func() error {
		rdbConn, err := GetRedisConn(ctx)
		if err != nil {
			return err
		}
		err = operation(rdbConn)
		if isConnectionError(err) {
			_ = reconnectRedis(ctx)
			return err
		}
		return err
	}
	err := retryOperation()
	if err != nil && isConnectionError(err) {
		return retrier.Retry(retryOperation, 2, 150*time.Millisecond, 2*time.Second)
	}
	return err
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.Nil) {
		return false
	}
	return err == redis.TxFailedErr || err.Error() == "redis: connection closed"
}

func MSet(ctx context.Context, values map[string]string) error {
	operation := func(client *redis.Client) error {
		var pairs []any
		for key, value := range values {
			pairs = append(pairs, key, value)
		}
		return client.MSet(ctx, pairs...).Err()
	}
	return executeWithRetry(ctx, operation)
}

func MSetObject(ctx context.Context, values map[string]any) error {
	operation := func(client *redis.Client) error {
		var pairs []any
		for key, value := range values {
			data, err := json.Marshal(value)
			if err != nil {
				log.Error().Err(err).Any("key", key).Any("Data", data).Msg("Error marshalling object")
				return err
			}
			pairs = append(pairs, key, string(data))
		}

		return client.MSet(ctx, pairs...).Err()
	}
	return executeWithRetry(ctx, operation)
}

func MSetObjectWithExp(ctx context.Context, values map[string]any, exp time.Duration) error {
	operation := func(client *redis.Client) error {
		var pairs []any
		for key, value := range values {
			data, jsonMarshalErr := json.Marshal(value)
			if jsonMarshalErr != nil {
				log.Error().Err(jsonMarshalErr).Msg("Error marshalling object")
				return jsonMarshalErr
			}
			pairs = append(pairs, key, string(data))
		}

		pipe := client.TxPipeline()
		pipe.MSet(ctx, pairs...)
		for key := range values {
			pipe.Expire(ctx, key, exp)
		}
		_, err := pipe.Exec(ctx)
		return err
	}
	return executeWithRetry(ctx, operation)
}

func Set(ctx context.Context, key string, value string, exp time.Duration) error {
	operation := func(client *redis.Client) error {
		return client.Set(ctx, key, value, exp).Err()
	}

	return executeWithRetry(ctx, operation)
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
	var result []any
	operation := func(client *redis.Client) error {
		data, err := client.MGet(ctx, keys...).Result()
		if err != nil {
			return err
		}
		result = data
		return nil
	}
	err := executeWithRetry(ctx, operation)
	return result, err
}

func MGetObject[T any](ctx context.Context, keys []string) ([]T, error) {
	results := []T{}

	if len(keys) == 0 {
		return results, nil
	}

	data, err := MGet(ctx, keys)
	if err != nil {
		log.Error().Strs("keys", keys).Err(err).Msg("Error getting objects from MGetObject")
		return results, err
	}

	for _, d := range data {
		if d == nil {
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
	var result string
	operation := func(client *redis.Client) error {
		data, err := client.Get(ctx, key).Result()
		if err != nil {
			return err
		}
		result = data
		return nil
	}
	err := executeWithRetry(ctx, operation)
	return result, err
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
	var result []string
	operation := func(client *redis.Client) error {
		data, err := client.LRange(ctx, key, start, end).Result()
		if err != nil {
			return err
		}
		result = data
		return nil
	}
	err := executeWithRetry(ctx, operation)
	return result, err
}

func LRangeObject[T any](ctx context.Context, key string, start int64, end int64) ([]T, error) {
	data, err := LRange(ctx, key, start, end)
	if err != nil {
		log.Error().Err(err).Msg("Error getting range")
		return nil, err
	}

	results := make([]T, len(data))
	for i, d := range data {
		err = json.Unmarshal([]byte(d), &results[i])
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling object")
			return nil, err
		}
	}
	return results, nil
}

func LPush(ctx context.Context, key string, values ...any) error {
	operation := func(client *redis.Client) error {
		return client.LPush(ctx, key, values...).Err()
	}
	return executeWithRetry(ctx, operation)
}

func LPushObject[T any](ctx context.Context, key string, values []T) error {
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
	var result []string
	operation := func(client *redis.Client) error {
		pipe := client.TxPipeline()
		lrange := pipe.LRange(ctx, key, 0, -1)
		pipe.Del(ctx, key)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
		result = lrange.Val()
		return nil
	}
	err := executeWithRetry(ctx, operation)
	return result, err
}

func PopAllObject[T any](ctx context.Context, key string) ([]T, error) {
	data, err := PopAll(ctx, key)
	if err != nil {
		log.Error().Err(err).Msg("Error popping all objects")
		return nil, err
	}

	results := make([]T, len(data))
	for i, d := range data {
		err = json.Unmarshal([]byte(d), &results[i])
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling object")
			return nil, err
		}
	}
	return results, nil
}

func connectToRedis(ctx context.Context, connectionInfo RedisConnectionInfo) (*redis.Client, error) {
	rdbConn := redis.NewClient(&redis.Options{
		Addr: connectionInfo.Host + ":" + connectionInfo.Port,
	})
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

func setRedis(ctx context.Context, rdb *redis.Client, key string, value string, exp time.Duration) error {
	return rdb.Set(ctx, key, value, exp).Err()
}

func getRedis(ctx context.Context, rdb *redis.Client, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
}

func CloseRedis() {
	if rdb != nil {
		rdb.Close()
		rdb = nil
	}
}
