package realtime

import (
	"fmt"
	"log/slog"

	"github.com/naralabs/naralabs-atlas/config"
)

func NewPublisher(cfg *config.Config) (Publisher, func() error, error) {
	if cfg == nil || !cfg.Realtime.Enabled {
		return NewNoopPublisher(), func() error { return nil }, nil
	}
	pub, err := NewRedisPublisher(cfg.Realtime.RedisURL, cfg.Realtime.IngestChannel)
	if err != nil {
		return nil, nil, fmt.Errorf("redis publisher: %w", err)
	}
	return pub, pub.Close, nil
}

func NewSubscriber(cfg *config.Config, log *slog.Logger) (*RedisSubscriber, error) {
	if cfg == nil || !cfg.Realtime.Enabled {
		return nil, nil
	}
	return NewRedisSubscriber(cfg.Realtime.RedisURL, cfg.Realtime.IngestChannel, log)
}
