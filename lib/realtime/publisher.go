package realtime

import "context"

// Publisher notifies the realtime layer that new events were ingested.
type Publisher interface {
	PublishIngest(ctx context.Context, msg IngestMessage) error
}
