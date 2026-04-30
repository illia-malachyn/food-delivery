//go:build integration

package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/persistence"
)

type OrderToOutboxIntegrationSuite struct {
	suite.Suite

	brokers    []string
	orderTopic string

	postgresContainer testcontainers.Container
	dbPool            *pgxpool.Pool

	consumer *infrastructure.OrderEventsConsumer
	ctx      context.Context
	cancel   context.CancelFunc
}

func TestOrderToOutboxIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OrderToOutboxIntegrationSuite))
}

func (s *OrderToOutboxIntegrationSuite) SetupSuite() {
	s.brokers = kafkaBrokersForTest(s.T())
	s.orderTopic = "order.events." + uuid.NewString()

	ctx := context.Background()
	require.NoError(s.T(), ensureTopicOrderOutbox(ctx, s.brokers[0], s.orderTopic))

	container, dbDSN := startPostgresForOutboxSuite(ctx, s.T())
	s.postgresContainer = container

	dbPool, err := pgxpool.New(ctx, dbDSN)
	require.NoError(s.T(), err)
	s.dbPool = dbPool
	require.NoError(s.T(), applyPaymentSchemaOutboxSuite(ctx, s.dbPool))
}

func (s *OrderToOutboxIntegrationSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.postgresContainer != nil {
		_ = s.postgresContainer.Terminate(context.Background())
	}
}

func (s *OrderToOutboxIntegrationSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	require.NoError(s.T(), truncatePaymentTablesOutboxSuite(s.ctx, s.dbPool))

	repository := persistence.NewPostgresPaymentRepository(s.dbPool)
	service := application.NewPaymentService(repository)

	s.consumer = infrastructure.NewOrderEventsConsumer(
		s.brokers,
		s.orderTopic,
		"payment-outbox-it-"+uuid.NewString(),
		service,
		repository,
		1000,
		"USD",
	)
	go s.consumer.Run(s.ctx)
}

func (s *OrderToOutboxIntegrationSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.consumer != nil {
		_ = s.consumer.Close()
	}
}

func (s *OrderToOutboxIntegrationSuite) TestOrderPlacedWritesPaymentConfirmedOutboxEvent() {
	orderID := uuid.NewString()
	payload, err := json.Marshal(map[string]any{
		"version":     2,
		"order_id":    orderID,
		"customer_id": "user-1",
		"item_id":     "item-1",
		"quantity":    2,
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		"source":      "order-service",
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEventOutboxSuite(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", payload))

	outbox := waitOutboxEvent(s.T(), s.dbPool, "PaymentConfirmed", orderID, 20*time.Second)
	assert.Equal(s.T(), "PaymentConfirmed", outbox.EventName)

	var body map[string]any
	require.NoError(s.T(), json.Unmarshal(outbox.Payload, &body))
	assert.Equal(s.T(), orderID, body["order_id"])
	assert.Equal(s.T(), float64(1000), body["amount"])
	assert.Equal(s.T(), "USD", body["currency"])
}

func (s *OrderToOutboxIntegrationSuite) TestOrderCancelledWritesPaymentRefundedOutboxEvent() {
	orderID := uuid.NewString()
	orderPlacedPayload, err := json.Marshal(map[string]any{
		"version":     2,
		"order_id":    orderID,
		"customer_id": "user-1",
		"item_id":     "item-1",
		"quantity":    1,
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		"source":      "order-service",
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEventOutboxSuite(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", orderPlacedPayload))
	_ = waitOutboxEvent(s.T(), s.dbPool, "PaymentConfirmed", orderID, 20*time.Second)

	orderCancelledPayload, err := json.Marshal(map[string]any{
		"version":     1,
		"order_id":    orderID,
		"reason":      "customer request",
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEventOutboxSuite(s.ctx, s.brokers, s.orderTopic, "OrderCancelled", orderCancelledPayload))

	outbox := waitOutboxEvent(s.T(), s.dbPool, "PaymentRefunded", orderID, 20*time.Second)
	assert.Equal(s.T(), "PaymentRefunded", outbox.EventName)
}

type outboxRow struct {
	EventName string
	Payload   []byte
}

func waitOutboxEvent(t *testing.T, db *pgxpool.Pool, eventName string, orderID string, timeout time.Duration) outboxRow {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var row outboxRow
		err := db.QueryRow(
			context.Background(),
			`SELECT event_name, payload
			 FROM payment_outbox
			 WHERE event_name = $1 AND payload->>'order_id' = $2
			 ORDER BY created_at DESC
			 LIMIT 1`,
			eventName,
			orderID,
		).Scan(&row.EventName, &row.Payload)
		if err == nil {
			return row
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for outbox event %s for order %s", eventName, orderID)
	return outboxRow{}
}

func publishOrderEventOutboxSuite(ctx context.Context, brokers []string, topic string, eventName string, payload []byte) error {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
		BatchTimeout:           10 * time.Millisecond,
	}
	defer writer.Close()

	return writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_name", Value: []byte(eventName)},
		},
		Time: time.Now().UTC(),
	})
}

func ensureTopicOrderOutbox(ctx context.Context, broker string, topic string) error {
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

func startPostgresForOutboxSuite(ctx context.Context, t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.3-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "payments_user",
				"POSTGRES_PASSWORD": "payments_password",
				"POSTGRES_DB":       "payments",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp"),
		},
		Started: true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dbDSN := fmt.Sprintf("postgres://payments_user:payments_password@%s:%s/payments?sslmode=disable", host, port.Port())
	return container, dbDSN
}

func applyPaymentSchemaOutboxSuite(ctx context.Context, db *pgxpool.Pool) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			order_id UUID NOT NULL,
			amount DECIMAL(10, 2) NOT NULL,
			currency VARCHAR(3) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			provider_transaction_id VARCHAR(255),
			paid_at TIMESTAMPTZ,
			failure_reason TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS payment_outbox (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			aggregate_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			event_name TEXT NOT NULL,
			event_version INT NOT NULL,
			payload JSONB NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func truncatePaymentTablesOutboxSuite(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `TRUNCATE TABLE payment_outbox, payments`)
	return err
}
