package messaging

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, event SiteCheckEvent) error
	Close() error
}
