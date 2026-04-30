//go:build integration

package order_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/infrastructure"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/persistence"
)

type OutboxKafkaIntegrationSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc

	db *pgxpool.Pool

	orderTopic string

	publisher *infrastructure.KafkaOutboxPublisher
	relay     *infrastructure.OutboxRelay
	reader    *kafka.Reader

	service *application.OrderService

	relayWG sync.WaitGroup
}

func TestOutboxKafkaIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OutboxKafkaIntegrationSuite))
}

func (s *OutboxKafkaIntegrationSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	db, err := pgxpool.New(s.ctx, integrationPostgresDSN)
	require.NoError(s.T(), err)
	s.db = db
	require.NoError(s.T(), truncateTestData(s.ctx, s.db))

	s.orderTopic = "order.events." + uuid.NewString()
	require.NoError(s.T(), ensureTopic(s.ctx, integrationKafkaBrokers[0], s.orderTopic))

	publisher, err := infrastructure.NewKafkaOutboxPublisher(integrationKafkaBrokers, s.orderTopic)
	require.NoError(s.T(), err)
	s.publisher = publisher

	s.relay = infrastructure.NewOutboxRelay(s.db, s.publisher, 50, 25*time.Millisecond)
	s.relayWG.Add(1)
	go func() {
		defer s.relayWG.Done()
		s.relay.Run(s.ctx)
	}()

	s.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  integrationKafkaBrokers,
		Topic:    s.orderTopic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	repo := persistence.NewPostgresOrderRepository(s.db)
	s.service = application.NewOrderService(repo, application.NewIntegrationEventUpcaster())
}

func (s *OutboxKafkaIntegrationSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
	s.relayWG.Wait()
	if s.reader != nil {
		_ = s.reader.Close()
	}
	if s.publisher != nil {
		_ = s.publisher.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *OutboxKafkaIntegrationSuite) TestCreateOrderPublishesOrderPlacedEventV2() {
	orderID, err := s.service.Create(s.ctx, &application.OrderDTO{
		UserId:   "user-1",
		ItemId:   "item-1",
		Quantity: 2,
	})
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), orderID)

	msg := readMessageWithTimeout(s.T(), s.reader, 20*time.Second)
	require.Equal(s.T(), "OrderPlaced", headerValue(msg.Headers, "event_name"))
	require.Equal(s.T(), "2", headerValue(msg.Headers, "event_version"))

	var payload map[string]any
	require.NoError(s.T(), json.Unmarshal(msg.Value, &payload))
	assert.Equal(s.T(), float64(2), payload["version"])
	assert.Equal(s.T(), orderID, payload["order_id"])
	assert.Equal(s.T(), "user-1", payload["customer_id"])
	assert.Equal(s.T(), "item-1", payload["item_id"])
	assert.Equal(s.T(), "order-service", payload["source"])
}

func (s *OutboxKafkaIntegrationSuite) TestCancelOrderPublishesOrderCancelledEvent() {
	orderID, err := s.service.Create(s.ctx, &application.OrderDTO{
		UserId:   "user-2",
		ItemId:   "item-9",
		Quantity: 1,
	})
	require.NoError(s.T(), err)

	_ = readMessageWithTimeout(s.T(), s.reader, 20*time.Second) // first OrderPlaced event

	require.NoError(s.T(), s.service.Cancel(s.ctx, orderID, "payment failed"))

	msg := readMessageWithTimeout(s.T(), s.reader, 20*time.Second)
	require.Equal(s.T(), "OrderCancelled", headerValue(msg.Headers, "event_name"))
	require.Equal(s.T(), "1", headerValue(msg.Headers, "event_version"))

	var payload map[string]any
	require.NoError(s.T(), json.Unmarshal(msg.Value, &payload))
	assert.Equal(s.T(), float64(1), payload["version"])
	assert.Equal(s.T(), orderID, payload["order_id"])
	assert.Equal(s.T(), "payment failed", payload["reason"])
}

func truncateTestData(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `TRUNCATE TABLE outbox, orders`) // plain truncate is enough for integration test isolation.
	return err
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
			// Wait until topic metadata and leader are visible to reduce early publish retries.
			readyErr := waitTopicReady(ctx, broker, topic)
			if readyErr == nil {
				return nil
			}
			lastErr = readyErr
			time.Sleep(500 * time.Millisecond)
			continue
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		return errors.New("failed to create topic")
	}
	return lastErr
}

func waitTopicReady(ctx context.Context, broker string, topic string) error {
	var lastErr error
	for i := 0; i < 30; i++ {
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}

		partitions, err := conn.ReadPartitions()
		_ = conn.Close()
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}

		hasReadyPartition := false
		for _, p := range partitions {
			if p.Topic == topic && p.ID == 0 {
				hasReadyPartition = true
				break
			}
		}
		if !hasReadyPartition {
			lastErr = fmt.Errorf("topic %s partition metadata not ready", topic)
			time.Sleep(250 * time.Millisecond)
			continue
		}

		leaderConn, err := kafka.DialLeader(ctx, "tcp", broker, topic, 0)
		if err == nil {
			_ = leaderConn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}

	if lastErr == nil {
		return fmt.Errorf("topic %s is not ready", topic)
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
