package persistence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/illia-malachyn/food-delivery/payment/domain"
)

func TestMapDomainEventToOutboxRecord_PaymentPaid(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-1", 1500, "USD")
	require.NoError(t, err)
	require.NoError(t, payment.MarkPaid())
	events := payment.FlushEvents()
	require.Len(t, events, 2)

	record, ok, err := mapDomainEventToOutboxRecord(payment, events[1])
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "payment", record.aggregateType)
	assert.Equal(t, "PaymentConfirmed", record.eventName)
	assert.Equal(t, 1, record.eventVersion)

	payload, payloadOk := record.payload.(map[string]any)
	require.True(t, payloadOk)
	assert.Equal(t, "order-1", payload["order_id"])
	assert.Equal(t, payment.ID(), payload["payment_id"])
	assert.Equal(t, int64(1500), payload["amount"])
	assert.Equal(t, "USD", payload["currency"])
	assert.NotEmpty(t, payload["occurred_at"])
}

func TestMapDomainEventToOutboxRecord_PaymentInitiatedIgnored(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-2", 500, "USD")
	require.NoError(t, err)
	events := payment.FlushEvents()
	require.Len(t, events, 1)

	_, ok, err := mapDomainEventToOutboxRecord(payment, events[0])
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMapDomainEventToOutboxRecord_UnknownType(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-3", 700, "USD")
	require.NoError(t, err)

	record, ok, err := mapDomainEventToOutboxRecord(payment, unknownDomainEvent{})
	require.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, outboxRecord{}, record)
}

type unknownDomainEvent struct{}

func (unknownDomainEvent) EventName() string { return "Unknown" }

func TestMapDomainEventToOutboxRecord_FailedAndRefunded(t *testing.T) {
	t.Parallel()

	failedPayment, err := domain.NewPayment("order-4", 900, "EUR")
	require.NoError(t, err)
	require.NoError(t, failedPayment.MarkFailed("provider_timeout"))
	failedEvents := failedPayment.FlushEvents()
	require.Len(t, failedEvents, 2)

	failedRecord, ok, err := mapDomainEventToOutboxRecord(failedPayment, failedEvents[1])
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "PaymentFailed", failedRecord.eventName)

	failedPayload, payloadOk := failedRecord.payload.(map[string]any)
	require.True(t, payloadOk)
	assert.Equal(t, "provider_timeout", failedPayload["reason"])

	refundPayment, err := domain.NewPayment("order-5", 1200, "USD")
	require.NoError(t, err)
	require.NoError(t, refundPayment.MarkPaid())
	require.NoError(t, refundPayment.Refund())
	refundEvents := refundPayment.FlushEvents()
	require.Len(t, refundEvents, 3)

	refundedRecord, ok, err := mapDomainEventToOutboxRecord(refundPayment, refundEvents[2])
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "PaymentRefunded", refundedRecord.eventName)
	assert.WithinDuration(t, time.Now().UTC(), refundedRecord.occurredAt.UTC(), 5*time.Second)
}
