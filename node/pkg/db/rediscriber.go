package db

import (
	"context"

	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Rediscriber struct {
	client        *redis.Client
	router        func(*redis.Message) error
	redisChannels []string
}

type RediscriberConfig struct {
	RedisHost     string
	RedisPort     string
	RedisChannels []string
	Router        func(*redis.Message) error
}

type RediscriberOption func(*RediscriberConfig)

func WithRedisHost(host string) RediscriberOption {
	return func(config *RediscriberConfig) {
		config.RedisHost = host
	}
}

func WithRedisPort(port string) RediscriberOption {
	return func(config *RediscriberConfig) {
		config.RedisPort = port
	}
}

func WithRedisChannels(channels []string) RediscriberOption {
	return func(config *RediscriberConfig) {
		config.RedisChannels = channels
	}
}

func WithRedisRouter(router func(*redis.Message) error) RediscriberOption {
	return func(config *RediscriberConfig) {
		config.Router = router
	}
}

func NewRediscriber(ctx context.Context, opts ...RediscriberOption) (*Rediscriber, error) {
	config := &RediscriberConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.Router == nil {
		return nil, errorsentinel.ErrRediscriberRouterNotFound
	}

	client, err := newRedisClient(ctx, config.RedisHost, config.RedisPort)
	if err != nil {
		return nil, err
	}
	return &Rediscriber{
		client:        client,
		router:        config.Router,
		redisChannels: config.RedisChannels,
	}, nil
}

func (r *Rediscriber) Start(ctx context.Context) error {
	err := r.client.Ping(ctx).Err()
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to redis")
		return err
	}

	for _, channel := range r.redisChannels {
		sub := r.client.Subscribe(ctx, channel)
		rawCh := sub.Channel()
		go func(channel string) {
			for {
				select {
				case <-ctx.Done():
					sub.Close()
					return
				case msg := <-rawCh:
					if ctx.Err() != nil {
						return
					}
					err := r.router(msg)
					if err != nil {
						log.Error().Err(err).Str("channel", channel).Msg("Error handling redis message")
					}
				}
			}
		}(channel)
	}
	return nil
}

func newRedisClient(ctx context.Context, host string, port string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: host + ":" + port,
	})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to redis")
		return nil, err
	}
	return client, nil
}
