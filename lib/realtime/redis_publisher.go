package realtime

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client  *redis.Client
	channel string
}

func NewRedisPublisher(redisURL, channel string) (*RedisPublisher, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisPublisher{client: client, channel: channel}, nil
}

func (p *RedisPublisher) PublishIngest(ctx context.Context, msg IngestMessage) error {
	payload, err := EncodeIngestMessage(msg)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, p.channel, payload).Err()
}

func (p *RedisPublisher) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}
