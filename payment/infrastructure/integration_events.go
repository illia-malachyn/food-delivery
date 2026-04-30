package infrastructure

import "time"

type PaymentConfirmedEvent struct {
	Version    int       `json:"version"`
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentConfirmedEvent) EventName() string {
	return "PaymentConfirmed"
}

type PaymentFailedEvent struct {
	Version    int       `json:"version"`
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentFailedEvent) EventName() string {
	return "PaymentFailed"
}

type PaymentRefundedEvent struct {
	Version    int       `json:"version"`
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentRefundedEvent) EventName() string {
	return "PaymentRefunded"
}
