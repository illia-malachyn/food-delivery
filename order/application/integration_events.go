package application

import "time"

type IntegrationEvent interface {
	EventName() string
	EventVersion() int
	AggregateID() string
	AggregateType() string
	EventOccurredAt() time.Time
}

type OrderPlacedEvent struct {
	Version    int       `json:"version"`
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	ItemID     string    `json:"item_id"`
	Quantity   uint      `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e OrderPlacedEvent) EventName() string {
	return "OrderPlacedEvent"
}

func (e OrderPlacedEvent) EventVersion() int {
	return e.Version
}

func (e OrderPlacedEvent) AggregateID() string {
	return e.OrderID
}

func (e OrderPlacedEvent) AggregateType() string {
	return "order"
}

func (e OrderPlacedEvent) EventOccurredAt() time.Time {
	return e.OccurredAt
}

type OrderPlacedEventV2 struct {
	Version    int       `json:"version"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	ItemID     string    `json:"item_id"`
	Quantity   uint      `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
	Source     string    `json:"source"`
}

func (e OrderPlacedEventV2) EventName() string {
	return "OrderPlacedEvent"
}

func (e OrderPlacedEventV2) EventVersion() int {
	return e.Version
}

func (e OrderPlacedEventV2) AggregateID() string {
	return e.OrderID
}

func (e OrderPlacedEventV2) AggregateType() string {
	return "order"
}

func (e OrderPlacedEventV2) EventOccurredAt() time.Time {
	return e.OccurredAt
}
