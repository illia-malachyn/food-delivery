package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/infrastructure"
	httpinfra "github.com/illia-malachyn/food-delivery/order/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/persistence"
)

func main() {
	log.Println("order microservice started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connPool, err := pgxpool.New(ctx, databaseURLFromEnv())
	if err != nil {
		log.Fatal(err)
	}
	defer connPool.Close()

	kafkaPublisher, err := infrastructure.NewKafkaOutboxPublisher(
		kafkaBrokersFromEnv(),
		getEnvOrDefault("KAFKA_TOPIC_ORDER_EVENTS", "order.events"),
	)
	if err != nil {
		log.Fatalf("cannot initialize kafka publisher: %v", err)
	}
	defer kafkaPublisher.Close()

	outboxRelay := infrastructure.NewOutboxRelay(
		connPool,
		kafkaPublisher,
		intFromEnv("OUTBOX_BATCH_SIZE", 100),
		durationFromEnv("OUTBOX_POLL_INTERVAL", 2*time.Second),
	)
	go outboxRelay.Run(ctx)

	postgresOrderRepository := persistence.NewPostgresOrderRepository(connPool)
	eventUpcaster := application.NewIntegrationEventUpcaster()
	orderService := application.NewOrderService(postgresOrderRepository, eventUpcaster)
	orderHandler := httpinfra.CreateOrderHandler(orderService)

	mux := http.NewServeMux()
	mux.Handle("/orders", orderHandler)

	server := &http.Server{
		Addr:    ":9876",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("order server shutdown failed: %v", shutdownErr)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func databaseURLFromEnv() string {
	dbUser := getEnvOrDefaultMany([]string{"ORDER_DB_USER", "DB_USER"}, "orders_user")
	dbPassword := getEnvOrDefaultMany([]string{"ORDER_DB_PASSWORD", "DB_PASSWORD"}, "orders_password")
	dbHost := getEnvOrDefaultMany([]string{"ORDER_DB_HOST", "DB_HOST"}, "localhost")
	dbPort := getEnvOrDefaultMany([]string{"ORDER_DB_PORT", "DB_PORT"}, "5432")
	dbName := getEnvOrDefaultMany([]string{"ORDER_DB_NAME", "DB_NAME"}, "orders")
	sslMode := getEnvOrDefaultMany([]string{"ORDER_DB_SSLMODE", "DB_SSL_MODE"}, "disable")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
		sslMode,
	)
}

func kafkaBrokersFromEnv() []string {
	raw := getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))

	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}

	if len(brokers) == 0 {
		return []string{"localhost:9092"}
	}

	return brokers
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration for %s=%q, using fallback %s", key, raw, fallback)
		return fallback
	}

	return parsed
}

func intFromEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid int for %s=%q, using fallback %d", key, raw, fallback)
		return fallback
	}

	return parsed
}

func getEnvOrDefaultMany(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
