package domain

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	id            string
	orderID       string
	amount        int64
	currency      string
	status        PaymentStatus
	failureReason string
	createdAt     time.Time
	events        []DomainEvent
}

func NewPayment(orderID string, amount int64, currency string) (*Payment, error) {
	normalizedOrderID, normalizedCurrency, err := validateNewPaymentInput(orderID, amount, currency)
	if err != nil {
		return nil, err
	}

	payment := &Payment{
		id:        uuid.NewString(),
		orderID:   normalizedOrderID,
		amount:    amount,
		currency:  normalizedCurrency,
		status:    PaymentStatusPending,
		createdAt: time.Now().UTC(),
	}

	payment.recordEvent(PaymentInitiatedEvent{
		EventBase: payment.eventBase(),
		Amount:    payment.amount,
		Currency:  payment.currency,
	})

	return payment, nil
}

func ReconstructPayment(
	id string,
	orderID string,
	amount int64,
	currency string,
	status PaymentStatus,
	failureReason string,
	createdAt time.Time,
) (*Payment, error) {
	normalizedID, normalizedOrderID, normalizedCurrency, normalizedFailureReason, err := validateReconstructedPayment(
		id,
		orderID,
		amount,
		currency,
		status,
		failureReason,
		createdAt,
	)
	if err != nil {
		return nil, err
	}

	return &Payment{
		id:            normalizedID,
		orderID:       normalizedOrderID,
		amount:        amount,
		currency:      normalizedCurrency,
		status:        status,
		failureReason: normalizedFailureReason,
		createdAt:     createdAt.UTC(),
	}, nil
}

func (p *Payment) MarkPaid() error {
	switch p.status {
	case PaymentStatusPaid:
		return nil
	case PaymentStatusPending:
		p.status = PaymentStatusPaid
		p.failureReason = ""
		p.recordEvent(PaymentPaidEvent{EventBase: p.eventBase()})
		return nil
	default:
		return ErrInvalidStateTransition
	}
}

func (p *Payment) MarkFailed(reason string) error {
	normalizedReason, err := validateFailureReason(reason)
	if err != nil {
		return err
	}

	switch p.status {
	case PaymentStatusFailed:
		if p.failureReason == normalizedReason {
			return nil
		}
		return ErrInvalidStateTransition
	case PaymentStatusPending:
		p.status = PaymentStatusFailed
		p.failureReason = normalizedReason
		p.recordEvent(PaymentFailedEvent{EventBase: p.eventBase(), Reason: normalizedReason})
		return nil
	default:
		return ErrInvalidStateTransition
	}
}

func (p *Payment) Refund() error {
	switch p.status {
	case PaymentStatusRefunded:
		return nil
	case PaymentStatusPaid:
		p.status = PaymentStatusRefunded
		p.recordEvent(PaymentRefundedEvent{EventBase: p.eventBase()})
		return nil
	default:
		return ErrInvalidStateTransition
	}
}

func (p *Payment) FlushEvents() []DomainEvent {
	events := p.events
	p.events = nil
	return events
}

func (p *Payment) ID() string            { return p.id }
func (p *Payment) OrderID() string       { return p.orderID }
func (p *Payment) Amount() int64         { return p.amount }
func (p *Payment) Currency() string      { return p.currency }
func (p *Payment) Status() PaymentStatus { return p.status }
func (p *Payment) FailureReason() string { return p.failureReason }
func (p *Payment) CreatedAt() time.Time  { return p.createdAt }

func (p *Payment) eventBase() EventBase {
	return EventBase{
		PaymentID:  p.id,
		OrderID:    p.orderID,
		OccurredAt: time.Now().UTC(),
	}
}

func (p *Payment) recordEvent(event DomainEvent) {
	p.events = append(p.events, event)
}
