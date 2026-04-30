package domain

import "time"

type DomainEvent interface {
	EventName() string
}

type EventBase struct {
	PaymentID  string
	OrderID    string
	OccurredAt time.Time
}

type PaymentInitiatedEvent struct {
	EventBase
	Amount   int64
	Currency string
}

func (e PaymentInitiatedEvent) EventName() string {
	return "PaymentInitiated"
}

type PaymentPaidEvent struct {
	EventBase
}

func (e PaymentPaidEvent) EventName() string {
	return "PaymentPaid"
}

type PaymentFailedEvent struct {
	EventBase
	Reason string
}

func (e PaymentFailedEvent) EventName() string {
	return "PaymentFailed"
}

type PaymentRefundedEvent struct {
	EventBase
}

func (e PaymentRefundedEvent) EventName() string {
	return "PaymentRefunded"
}
