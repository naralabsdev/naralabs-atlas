package realtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type IngestHandler func(ctx context.Context, msg IngestMessage) error

type RedisSubscriber struct {
	client  *redis.Client
	channel string
	log     *slog.Logger
}

func NewRedisSubscriber(redisURL, channel string, log *slog.Logger) (*RedisSubscriber, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	if log == nil {
		log = slog.Default()
	}
	return &RedisSubscriber{client: client, channel: channel, log: log}, nil
}

func (s *RedisSubscriber) Run(ctx context.Context, handler IngestHandler) error {
	pubsub := s.client.Subscribe(ctx, s.channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			ingest, err := DecodeIngestMessage([]byte(msg.Payload))
			if err != nil {
				s.log.Warn("realtime ingest decode failed", "error", err)
				continue
			}
			if handler == nil {
				continue
			}
			if err := handler(ctx, ingest); err != nil {
				s.log.Warn("realtime ingest handler failed", "error", err)
			}
		}
	}
}

func (s *RedisSubscriber) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
