package domain

import (
	"time"
)

type domainEvent interface {
	EventName() string
}

type EventBase struct {
	OrderID    string
	OccurredAt time.Time
}

type OrderPlacedEvent struct {
	EventBase
}

func (e OrderPlacedEvent) EventName() string {
	return "OrderPlaced"
}
