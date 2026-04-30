//go:build integration

package order_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	kafkamodule "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
)

var integrationKafkaContainer *kafkamodule.KafkaContainer
var integrationKafkaBrokers []string

var integrationPostgresContainer testcontainers.Container
var integrationPostgresDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	kafkaContainer, err := kafkamodule.RunContainer(ctx)
	if err != nil {
		log.Printf("failed to start kafka container: %v", err)
		os.Exit(1)
	}
	integrationKafkaContainer = kafkaContainer

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil || len(brokers) == 0 {
		log.Printf("failed to get kafka brokers: %v", err)
		_ = kafkaContainer.Terminate(context.Background())
		os.Exit(1)
	}
	integrationKafkaBrokers = brokers

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.3-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "orders",
				"POSTGRES_USER":     "orders_user",
				"POSTGRES_PASSWORD": "orders_password",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp"),
		},
		Started: true,
	})
	if err != nil {
		log.Printf("failed to start postgres container: %v", err)
		_ = kafkaContainer.Terminate(context.Background())
		os.Exit(1)
	}
	integrationPostgresContainer = postgresContainer

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		log.Printf("failed to get postgres host: %v", err)
		teardownContainers()
		os.Exit(1)
	}
	port, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		log.Printf("failed to get postgres port: %v", err)
		teardownContainers()
		os.Exit(1)
	}

	integrationPostgresDSN = fmt.Sprintf("postgres://orders_user:orders_password@%s:%s/orders?sslmode=disable", host, port.Port())

	db, err := pgxpool.New(ctx, integrationPostgresDSN)
	if err != nil {
		log.Printf("failed to open postgres pool: %v", err)
		teardownContainers()
		os.Exit(1)
	}
	if err := applySchema(ctx, db); err != nil {
		db.Close()
		log.Printf("failed to apply schema: %v", err)
		teardownContainers()
		os.Exit(1)
	}
	db.Close()

	exitCode := m.Run()
	teardownContainers()
	os.Exit(exitCode)
}

func teardownContainers() {
	if integrationPostgresContainer != nil {
		_ = integrationPostgresContainer.Terminate(context.Background())
	}
	if integrationKafkaContainer != nil {
		_ = integrationKafkaContainer.Terminate(context.Background())
	}
}

func applySchema(ctx context.Context, db *pgxpool.Pool) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			item_id TEXT NOT NULL,
			quantity BIGINT NOT NULL CHECK (quantity > 0),
			status TEXT NOT NULL CHECK (status IN ('draft', 'placed', 'confirmed', 'cancelled')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS outbox (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			aggregate_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			event_name TEXT NOT NULL,
			event_version INT NOT NULL,
			payload JSONB NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			published_at TIMESTAMPTZ NULL,
			retry_count INT NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_unpublished_created_at
			ON outbox (created_at)
			WHERE published_at IS NULL`,
	}

	for _, query := range queries {
		if _, err := db.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
