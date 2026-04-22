package infrastructure

import (
	"context"
	"log"

	"github.com/illia-malachyn/food-delivery/order/application"
)

type LogEventPublisher struct{}

var _ application.EventPublisher = (*LogEventPublisher)(nil)

func NewLogEventPublisher() *LogEventPublisher {
	return &LogEventPublisher{}
}

func (p *LogEventPublisher) Publish(_ context.Context, events []application.IntegrationEvent) error {
	for _, event := range events {
		log.Printf("published integration event: %s v%d %#v", event.EventName(), event.EventVersion(), event)
	}

	return nil
}
