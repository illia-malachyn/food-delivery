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

type OrderConfirmedEvent struct {
	EventBase
}

func (e OrderConfirmedEvent) EventName() string {
	return "OrderConfirmed"
}

type OrderCancelledEvent struct {
	EventBase
	Reason string
}

func (e OrderCancelledEvent) EventName() string {
	return "OrderCancelled"
}
