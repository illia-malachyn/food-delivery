//go:build integration

package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure"
)

type inMemoryPaymentRepository struct {
	mu      sync.Mutex
	byID    map[string]*domain.Payment
	byOrder map[string]string
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

type KafkaIntegrationSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc

	brokers []string

	orderTopic   string
	paymentTopic string

	repo      *inMemoryPaymentRepository
	service   *application.PaymentService
	publisher *infrastructure.KafkaPaymentEventPublisher
	consumer  *infrastructure.OrderEventsConsumer
	reader    *kafka.Reader
}

func TestKafkaIntegrationSuite(t *testing.T) {
	suite.Run(t, new(KafkaIntegrationSuite))
}

func (s *KafkaIntegrationSuite) SetupTest() {
	s.brokers = kafkaBrokersForTest(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.orderTopic = "order.events." + uuid.NewString()
	s.paymentTopic = "payment.events." + uuid.NewString()
	require.NoError(s.T(), ensureTopic(s.ctx, s.brokers[0], s.orderTopic))
	require.NoError(s.T(), ensureTopic(s.ctx, s.brokers[0], s.paymentTopic))

	s.repo = newInMemoryPaymentRepository()
	s.service = application.NewPaymentService(s.repo)

	publisher, err := infrastructure.NewKafkaPaymentEventPublisher(s.brokers, s.paymentTopic)
	require.NoError(s.T(), err)
	s.publisher = publisher

	s.consumer = infrastructure.NewOrderEventsConsumer(
		s.brokers,
		s.orderTopic,
		"payment-integration-"+uuid.NewString(),
		s.service,
		s.repo,
		s.publisher,
		1000,
		"USD",
	)
	go s.consumer.Run(s.ctx)

	s.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  s.brokers,
		Topic:    s.paymentTopic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
}

func (s *KafkaIntegrationSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.reader != nil {
		_ = s.reader.Close()
	}
	if s.consumer != nil {
		_ = s.consumer.Close()
	}
	if s.publisher != nil {
		_ = s.publisher.Close()
	}
}

func (s *KafkaIntegrationSuite) TestOrderPlacedProducesPaymentConfirmed() {
	payload, err := json.Marshal(map[string]any{
		"version":     2,
		"order_id":    "order-placed-1",
		"customer_id": "user-1",
		"item_id":     "item-1",
		"quantity":    2,
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		"source":      "order-service",
	})
	require.NoError(s.T(), err)

	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", payload))

	msg := readMessageWithTimeout(s.T(), s.reader, 15*time.Second)
	require.Equal(s.T(), "PaymentConfirmed", headerValue(msg.Headers, "event_name"))

	var confirmed infrastructure.PaymentConfirmedEvent
	require.NoError(s.T(), json.Unmarshal(msg.Value, &confirmed))
	assert.Equal(s.T(), "order-placed-1", confirmed.OrderID)
	assert.Equal(s.T(), int64(1000), confirmed.Amount)
	assert.Equal(s.T(), "USD", confirmed.Currency)
}

func (s *KafkaIntegrationSuite) TestOrderCancelledProducesPaymentRefunded() {
	orderPlacedPayload, err := json.Marshal(map[string]any{
		"version":     2,
		"order_id":    "order-cancel-1",
		"customer_id": "user-1",
		"item_id":     "item-1",
		"quantity":    1,
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		"source":      "order-service",
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", orderPlacedPayload))

	_ = readMessageWithTimeout(s.T(), s.reader, 15*time.Second) // PaymentConfirmed

	orderCancelledPayload, err := json.Marshal(map[string]any{
		"version":     1,
		"order_id":    "order-cancel-1",
		"reason":      "customer request",
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderCancelled", orderCancelledPayload))

	msg := readMessageWithTimeout(s.T(), s.reader, 15*time.Second)
	require.Equal(s.T(), "PaymentRefunded", headerValue(msg.Headers, "event_name"))

	var refunded infrastructure.PaymentRefundedEvent
	require.NoError(s.T(), json.Unmarshal(msg.Value, &refunded))
	assert.Equal(s.T(), "order-cancel-1", refunded.OrderID)
	assert.Equal(s.T(), int64(1000), refunded.Amount)
	assert.Equal(s.T(), "USD", refunded.Currency)
}

func publishOrderEvent(ctx context.Context, brokers []string, topic string, eventName string, payload []byte) error {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
		BatchTimeout:           10 * time.Millisecond,
	}
	defer writer.Close()

	message := kafka.Message{
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_name", Value: []byte(eventName)},
		},
		Time: time.Now().UTC(),
	}

	return writer.WriteMessages(ctx, message)
}

func ensureTopic(ctx context.Context, broker string, topic string) error {
	var lastErr error
	for i := 0; i < 30; i++ {
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		_ = conn.Close()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		return errors.New("failed to create topic")
	}
	return lastErr
}

func readMessageWithTimeout(t *testing.T, reader *kafka.Reader, timeout time.Duration) kafka.Message {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	require.NoError(t, err)
	return msg
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

func kafkaBrokersForTest(t *testing.T) []string {
	t.Helper()
	require.NotEmpty(t, integrationKafkaBrokers)
	return integrationKafkaBrokers
}
