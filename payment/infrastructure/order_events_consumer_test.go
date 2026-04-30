package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
)

type inMemoryPaymentRepository struct {
	mu         sync.Mutex
	byID       map[string]*domain.Payment
	byOrder    map[string]string
	savedEvent []domain.DomainEvent
}

func newInMemoryPaymentRepository() *inMemoryPaymentRepository {
	return &inMemoryPaymentRepository{
		byID:    map[string]*domain.Payment{},
		byOrder: map[string]string{},
	}
}

func (r *inMemoryPaymentRepository) GetByID(_ context.Context, id string) (*domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	payment, ok := r.byID[id]
	if !ok {
		return nil, application.ErrPaymentNotFound
	}
	return payment, nil
}

func (r *inMemoryPaymentRepository) GetByOrderID(_ context.Context, orderID string) (*domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byOrder[orderID]
	if !ok {
		return nil, application.ErrPaymentNotFound
	}
	payment, ok := r.byID[id]
	if !ok {
		return nil, application.ErrPaymentNotFound
	}
	return payment, nil
}

func (r *inMemoryPaymentRepository) Save(_ context.Context, payment *domain.Payment, _ []domain.DomainEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[payment.ID()] = payment
	r.byOrder[payment.OrderID()] = payment.ID()
	return nil
}

func TestOrderEventsConsumer_HandleOrderPlacedMarksPaymentPaid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryPaymentRepository()
	service := application.NewPaymentService(repo)
	consumer := NewOrderEventsConsumer(
		[]string{"localhost:9092"},
		"order.events",
		"payment-service-test",
		service,
		repo,
		1000,
		"USD",
	)
	defer consumer.Close()

	payload, err := json.Marshal(OrderPlacedEvent{Version: 2, OrderID: "order-1"})
	require.NoError(t, err)

	err = consumer.HandleMessage(context.Background(), kafka.Message{
		Value:   payload,
		Headers: []kafka.Header{{Key: "event_name", Value: []byte("OrderPlaced")}},
	})
	require.NoError(t, err)

	payment, err := repo.GetByOrderID(context.Background(), "order-1")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusPaid, payment.Status())
}

func TestOrderEventsConsumer_HandleOrderCancelledMarksPaymentRefunded(t *testing.T) {
	t.Parallel()

	repo := newInMemoryPaymentRepository()
	service := application.NewPaymentService(repo)
	consumer := NewOrderEventsConsumer(
		[]string{"localhost:9092"},
		"order.events",
		"payment-service-test",
		service,
		repo,
		1000,
		"USD",
	)
	defer consumer.Close()

	payment, err := domain.NewPayment("order-2", 1000, "USD")
	require.NoError(t, err)
	require.NoError(t, payment.MarkPaid())
	require.NoError(t, repo.Save(context.Background(), payment, payment.FlushEvents()))

	payload, err := json.Marshal(OrderCancelledEvent{Version: 1, OrderID: "order-2", Reason: "cancelled"})
	require.NoError(t, err)

	err = consumer.HandleMessage(context.Background(), kafka.Message{
		Value:   payload,
		Headers: []kafka.Header{{Key: "event_name", Value: []byte("OrderCancelled")}},
	})
	require.NoError(t, err)

	updated, err := repo.GetByOrderID(context.Background(), "order-2")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusRefunded, updated.Status())
}

func TestOrderEventsConsumer_IgnoresUnknownEvent(t *testing.T) {
	t.Parallel()

	repo := newInMemoryPaymentRepository()
	service := application.NewPaymentService(repo)
	consumer := NewOrderEventsConsumer(
		[]string{"localhost:9092"},
		"order.events",
		"payment-service-test",
		service,
		repo,
		1000,
		"USD",
	)
	defer consumer.Close()

	err := consumer.HandleMessage(context.Background(), kafka.Message{
		Value:   []byte(`{"foo":"bar"}`),
		Headers: []kafka.Header{{Key: "event_name", Value: []byte("SomethingElse")}},
	})
	require.NoError(t, err)
}

func TestOrderEventsConsumer_ReturnsDecodeErrorForBrokenPayload(t *testing.T) {
	t.Parallel()

	repo := newInMemoryPaymentRepository()
	service := application.NewPaymentService(repo)
	consumer := NewOrderEventsConsumer(
		[]string{"localhost:9092"},
		"order.events",
		"payment-service-test",
		service,
		repo,
		1000,
		"USD",
	)
	defer consumer.Close()

	err := consumer.HandleMessage(context.Background(), kafka.Message{
		Value:   []byte("{"),
		Headers: []kafka.Header{{Key: "event_name", Value: []byte("OrderPlaced")}},
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, application.ErrPaymentNotFound))
}
