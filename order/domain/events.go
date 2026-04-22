package domain

import (
	"time"
)

type DomainEvent interface {
	EventName() string
}

type EventBase struct {
	OrderID    string
	OccurredAt time.Time
}

type OrderPlacedEvent struct {
	EventBase
	UserID   string
	ItemID   string
	Quantity uint
}

func (e OrderPlacedEvent) EventName() string {
	return "OrderPlaced"
}
