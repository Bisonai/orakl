package db

import (
	"context"
	"sync"
	"time"

	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Rediscribe struct {
	client *redis.Client
	router func(context.Context, *redis.Message) error
	topics []string

	host              string
	port              string
	reconnectInterval time.Duration

	mu sync.Mutex
}

type RediscribeConfig struct {
	RedisHost         string
	RedisPort         string
	RedisTopics       []string
	Router            func(context.Context, *redis.Message) error
	ReconnectInterval time.Duration
}

type RediscribeOption func(*RediscribeConfig)

const (
	DefaultReconnectInterval   = 5 * time.Second
	DefaultRedisSubChannelSize = 1000
)

func WithRedisHost(host string) RediscribeOption {
	return func(config *RediscribeConfig) {
		config.RedisHost = host
	}
}

func WithRedisPort(port string) RediscribeOption {
	return func(config *RediscribeConfig) {
		config.RedisPort = port
	}
}

func WithRedisTopics(topics []string) RediscribeOption {
	return func(config *RediscribeConfig) {
		config.RedisTopics = topics
	}
}

func WithRedisRouter(router func(context.Context, *redis.Message) error) RediscribeOption {
	return func(config *RediscribeConfig) {
		config.Router = router
	}
}

func WithReconnectInterval(interval time.Duration) RediscribeOption {
	return func(config *RediscribeConfig) {
		config.ReconnectInterval = interval
	}
}

func NewRediscribe(ctx context.Context, opts ...RediscribeOption) (*Rediscribe, error) {
	config := &RediscribeConfig{
		ReconnectInterval: DefaultReconnectInterval,
	}
	for _, opt := range opts {
		opt(config)
	}

	if config.Router == nil {
		return nil, errorsentinel.ErrRediscriberRouterNotFound
	}

	return &Rediscribe{
		router: config.Router,
		topics: config.RedisTopics,

		host: config.RedisHost,
		port: config.RedisPort,

		reconnectInterval: config.ReconnectInterval,
	}, nil
}

func (r *Rediscribe) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := r.reconnect(ctx); err != nil {
				log.Error().Err(err).Msg("failed to establish initial connection")
				return
			}

			wg := new(sync.WaitGroup)
			wg.Add(len(r.topics))

			for _, topic := range r.topics {
				go r.subscribe(ctx, topic, wg)
			}

			wg.Wait()

			if r.client != nil {
				r.mu.Lock()
				r.client.Close()
				r.client = nil
				r.mu.Unlock()
			}

			time.Sleep(r.reconnectInterval)
		}
	}
}

func (r *Rediscribe) connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client != nil {
		_ = r.client.Close()
	}

	client, err := newRedisClient(ctx, r.host, r.port)
	if err != nil {
		return err
	}
	r.client = client
	return nil
}

func (r *Rediscribe) reconnect(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := r.connect(ctx)
			if err != nil {
				if isConnectionError(err) {
					time.Sleep(r.reconnectInterval)
					continue
				}
				return err
			}
			return nil
		}

	}
}

func (r *Rediscribe) subscribe(ctx context.Context, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	sub := r.client.Subscribe(ctx, topic)
	defer sub.Close()

	ch := sub.Channel(redis.WithChannelSize(DefaultRedisSubChannelSize))

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg != nil {
				if err := r.router(ctx, msg); err != nil {
					log.Error().Err(err).Str("channel", topic).Msg("Error handling redis message")
				}
			}
		}
	}
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
