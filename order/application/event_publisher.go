package application

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, events []IntegrationEvent) error
}
