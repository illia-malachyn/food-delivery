package persistence

import (
	"fmt"
	"time"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type outboxRecord struct {
	aggregateType string
	aggregateID   string
	eventName     string
	eventVersion  int
	payload       any
	occurredAt    time.Time
}

func mapDomainEventToOutboxRecord(payment *domain.Payment, event domain.DomainEvent) (outboxRecord, bool, error) {
	switch typed := event.(type) {
	case domain.PaymentPaidEvent:
		return outboxRecord{
			aggregateType: "payment",
			aggregateID:   typed.PaymentID,
			eventName:     "PaymentConfirmed",
			eventVersion:  1,
			occurredAt:    typed.OccurredAt,
			payload: map[string]any{
				"version":     1,
				"order_id":    typed.OrderID,
				"payment_id":  typed.PaymentID,
				"amount":      payment.Amount(),
				"currency":    payment.Currency(),
				"occurred_at": typed.OccurredAt.UTC().Format(time.RFC3339Nano),
			},
		}, true, nil
	case domain.PaymentFailedEvent:
		return outboxRecord{
			aggregateType: "payment",
			aggregateID:   typed.PaymentID,
			eventName:     "PaymentFailed",
			eventVersion:  1,
			occurredAt:    typed.OccurredAt,
			payload: map[string]any{
				"version":     1,
				"order_id":    typed.OrderID,
				"payment_id":  typed.PaymentID,
				"reason":      typed.Reason,
				"occurred_at": typed.OccurredAt.UTC().Format(time.RFC3339Nano),
			},
		}, true, nil
	case domain.PaymentRefundedEvent:
		return outboxRecord{
			aggregateType: "payment",
			aggregateID:   typed.PaymentID,
			eventName:     "PaymentRefunded",
			eventVersion:  1,
			occurredAt:    typed.OccurredAt,
			payload: map[string]any{
				"version":     1,
				"order_id":    typed.OrderID,
				"payment_id":  typed.PaymentID,
				"amount":      payment.Amount(),
				"currency":    payment.Currency(),
				"occurred_at": typed.OccurredAt.UTC().Format(time.RFC3339Nano),
			},
		}, true, nil
	case domain.PaymentInitiatedEvent:
		return outboxRecord{}, false, nil
	default:
		return outboxRecord{}, false, fmt.Errorf("unsupported domain event type %T", event)
	}
}
