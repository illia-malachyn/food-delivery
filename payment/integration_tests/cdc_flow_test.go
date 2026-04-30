//go:build integration

package payment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
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

type DebeziumCDCIntegrationSuite struct {
	suite.Suite

	brokers      []string
	orderTopic   string
	paymentTopic string

	postgresContainer testcontainers.Container
	connectContainer  testcontainers.Container
	dbPool            *pgxpool.Pool

	consumer *infrastructure.OrderEventsConsumer
	reader   *kafka.Reader

	ctx    context.Context
	cancel context.CancelFunc
}

func TestDebeziumCDCIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DebeziumCDCIntegrationSuite))
}

func (s *DebeziumCDCIntegrationSuite) SetupSuite() {
	if os.Getenv("RUN_PAYMENT_CDC_INTEGRATION") != "1" {
		s.T().Skip("set RUN_PAYMENT_CDC_INTEGRATION=1 to run Debezium CDC integration suite")
	}

	if runtime.GOOS != "linux" {
		s.T().Skip("Debezium CDC integration suite requires Linux docker networking")
	}

	s.brokers = kafkaBrokersForTest(s.T())
	s.orderTopic = "order.events." + uuid.NewString()
	s.paymentTopic = "payment.events." + uuid.NewString()

	ctx := context.Background()
	require.NoError(s.T(), ensureTopic(ctx, s.brokers[0], s.orderTopic))
	require.NoError(s.T(), ensureTopic(ctx, s.brokers[0], s.paymentTopic))

	postgres, dbDSN, dbHostPort := startPostgresContainer(ctx, s.T())
	s.postgresContainer = postgres

	dbPool, err := pgxpool.New(ctx, dbDSN)
	require.NoError(s.T(), err)
	s.dbPool = dbPool
	require.NoError(s.T(), applyPaymentSchema(ctx, s.dbPool))

	connectContainer, connectURL := startDebeziumConnectContainer(ctx, s.T(), s.brokers[0])
	s.connectContainer = connectContainer

	require.NoError(s.T(), waitConnectReady(ctx, connectURL))
	require.NoError(s.T(), registerOutboxConnector(ctx, connectURL, dbHostPort, s.paymentTopic))
	require.NoError(s.T(), waitConnectorRunning(ctx, connectURL, "payment-outbox-connector"))
}

func (s *DebeziumCDCIntegrationSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.connectContainer != nil {
		_ = s.connectContainer.Terminate(context.Background())
	}
	if s.postgresContainer != nil {
		_ = s.postgresContainer.Terminate(context.Background())
	}
}

func (s *DebeziumCDCIntegrationSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	require.NoError(s.T(), truncatePaymentTables(s.ctx, s.dbPool))

	repository := persistence.NewPostgresPaymentRepository(s.dbPool)
	service := application.NewPaymentService(repository)

	s.consumer = infrastructure.NewOrderEventsConsumer(
		s.brokers,
		s.orderTopic,
		"payment-cdc-it-"+uuid.NewString(),
		service,
		repository,
		1000,
		"USD",
	)
	go s.consumer.Run(s.ctx)

	s.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     s.brokers,
		Topic:       s.paymentTopic,
		GroupID:     "payment-cdc-reader-" + uuid.NewString(),
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
}

func (s *DebeziumCDCIntegrationSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.reader != nil {
		_ = s.reader.Close()
	}
	if s.consumer != nil {
		_ = s.consumer.Close()
	}
}

func (s *DebeziumCDCIntegrationSuite) TestOrderPlacedEmitsPaymentConfirmedViaCDC() {
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

	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", payload))

	msg := readMessageWithTimeout(s.T(), s.reader, 30*time.Second)
	require.Equal(s.T(), "PaymentConfirmed", headerValue(msg.Headers, "event_name"))

	var event map[string]any
	require.NoError(s.T(), json.Unmarshal(msg.Value, &event))
	assert.Equal(s.T(), orderID, event["order_id"])
	assert.Equal(s.T(), float64(1000), event["amount"])
	assert.Equal(s.T(), "USD", event["currency"])
}

func (s *DebeziumCDCIntegrationSuite) TestOrderCancelledEmitsPaymentRefundedViaCDC() {
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
	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderPlaced", orderPlacedPayload))

	first := readMessageWithTimeout(s.T(), s.reader, 30*time.Second)
	require.Equal(s.T(), "PaymentConfirmed", headerValue(first.Headers, "event_name"))

	orderCancelledPayload, err := json.Marshal(map[string]any{
		"version":     1,
		"order_id":    orderID,
		"reason":      "customer request",
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	require.NoError(s.T(), err)
	require.NoError(s.T(), publishOrderEvent(s.ctx, s.brokers, s.orderTopic, "OrderCancelled", orderCancelledPayload))

	refund := readMessageWithTimeout(s.T(), s.reader, 30*time.Second)
	require.Equal(s.T(), "PaymentRefunded", headerValue(refund.Headers, "event_name"))

	var event map[string]any
	require.NoError(s.T(), json.Unmarshal(refund.Value, &event))
	assert.Equal(s.T(), orderID, event["order_id"])
	assert.Equal(s.T(), float64(1000), event["amount"])
	assert.Equal(s.T(), "USD", event["currency"])
}

func publishOrderEvent(ctx context.Context, brokers []string, topic string, eventName string, payload []byte) error {
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
		if strings.EqualFold(header.Key, key) {
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

func startPostgresContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string, string) {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.3-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "postgres",
			},
			Cmd: []string{
				"postgres",
				"-c", "wal_level=logical",
				"-c", "max_replication_slots=10",
				"-c", "max_wal_senders=10",
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

	adminDSN := fmt.Sprintf("postgres://postgres:postgres@%s:%s/postgres?sslmode=disable", host, port.Port())
	adminPool, err := pgxpool.New(ctx, adminDSN)
	require.NoError(t, err)
	defer adminPool.Close()

	_, err = adminPool.Exec(ctx, "CREATE ROLE payments_user WITH LOGIN PASSWORD 'payments_password'")
	require.NoError(t, err)
	_, err = adminPool.Exec(ctx, "ALTER ROLE payments_user WITH REPLICATION")
	require.NoError(t, err)
	_, err = adminPool.Exec(ctx, "CREATE DATABASE payments OWNER payments_user")
	require.NoError(t, err)

	dbDSN := fmt.Sprintf("postgres://payments_user:payments_password@%s:%s/payments?sslmode=disable", host, port.Port())
	return container, dbDSN, fmt.Sprintf("%s:%s", host, port.Port())
}

func applyPaymentSchema(ctx context.Context, db *pgxpool.Pool) error {
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

func truncatePaymentTables(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `TRUNCATE TABLE payment_outbox, payments`)
	return err
}

func startDebeziumConnectContainer(ctx context.Context, t *testing.T, brokerHostPort string) (testcontainers.Container, string) {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "debezium/connect:2.6",
			Env: map[string]string{
				"BOOTSTRAP_SERVERS":                    brokerHostPort,
				"GROUP_ID":                             "payment-cdc-it-connect",
				"CONFIG_STORAGE_TOPIC":                 "payment-cdc-it-connect-config",
				"OFFSET_STORAGE_TOPIC":                 "payment-cdc-it-connect-offset",
				"STATUS_STORAGE_TOPIC":                 "payment-cdc-it-connect-status",
				"CONFIG_STORAGE_REPLICATION_FACTOR":    "1",
				"OFFSET_STORAGE_REPLICATION_FACTOR":    "1",
				"STATUS_STORAGE_REPLICATION_FACTOR":    "1",
				"KEY_CONVERTER":                        "org.apache.kafka.connect.storage.StringConverter",
				"VALUE_CONVERTER":                      "org.apache.kafka.connect.json.JsonConverter",
				"VALUE_CONVERTER_SCHEMAS_ENABLE":       "false",
				"CONNECT_KEY_CONVERTER_SCHEMAS_ENABLE": "false",
			},
			HostConfigModifier: func(hostConfig *dockercontainer.HostConfig) {
				hostConfig.NetworkMode = "host"
			},
		},
		Started: true,
	})
	require.NoError(t, err)

	return container, "http://localhost:8083"
}

func registerOutboxConnector(ctx context.Context, connectURL string, dbHostPort string, paymentTopic string) error {
	body := map[string]any{
		"name": "payment-outbox-connector",
		"config": map[string]string{
			"connector.class":                             "io.debezium.connector.postgresql.PostgresConnector",
			"plugin.name":                                 "pgoutput",
			"database.hostname":                           "localhost",
			"database.port":                               portOnly(dbHostPort),
			"database.user":                               "payments_user",
			"database.password":                           "payments_password",
			"database.dbname":                             "payments",
			"topic.prefix":                                "payment-cdc",
			"publication.name":                            "payment_outbox_publication",
			"publication.autocreate.mode":                 "filtered",
			"slot.name":                                   "payment_outbox_slot",
			"slot.drop.on.stop":                           "true",
			"table.include.list":                          "public.payment_outbox",
			"tombstones.on.delete":                        "false",
			"heartbeat.interval.ms":                       "10000",
			"key.converter":                               "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                             "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable":              "false",
			"transforms":                                  "outbox",
			"transforms.outbox.type":                      "io.debezium.transforms.outbox.EventRouter",
			"transforms.outbox.route.topic.replacement":   paymentTopic,
			"transforms.outbox.table.field.event.id":      "id",
			"transforms.outbox.table.field.event.key":     "aggregate_id",
			"transforms.outbox.table.field.event.payload": "payload",
			"transforms.outbox.table.fields.additional.placement": "event_name:header:event_name,event_version:header:event_version,aggregate_type:header:aggregate_type,aggregate_id:header:aggregate_id,occurred_at:header:occurred_at",
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, connectURL+"/connectors", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	return fmt.Errorf("register connector failed with status %d", resp.StatusCode)
}

func waitConnectReady(ctx context.Context, connectURL string) error {
	for i := 0; i < 120; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectURL+"/connectors", nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("kafka connect is not ready at %s", connectURL)
}

func waitConnectorRunning(ctx context.Context, connectURL string, name string) error {
	for i := 0; i < 60; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectURL+"/connectors/"+name+"/status", nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				var status map[string]any
				_ = json.NewDecoder(resp.Body).Decode(&status)
				_ = resp.Body.Close()

				connector, _ := status["connector"].(map[string]any)
				if strings.EqualFold(fmt.Sprint(connector["state"]), "RUNNING") {
					return nil
				}
			} else {
				_ = resp.Body.Close()
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("connector %s did not reach RUNNING state", name)
}

func portOnly(hostPort string) string {
	parts := strings.Split(hostPort, ":")
	return parts[len(parts)-1]
}
