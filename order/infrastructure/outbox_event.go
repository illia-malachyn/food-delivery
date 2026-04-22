package infrastructure

import "time"

// OutboxEvent represents a persisted integration event row from the outbox table.
type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventName     string
	EventVersion  int
	Payload       []byte
	OccurredAt    time.Time
}
