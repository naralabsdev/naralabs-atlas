package realtime

import "context"

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (NoopPublisher) PublishIngest(_ context.Context, _ IngestMessage) error {
	return nil
}
