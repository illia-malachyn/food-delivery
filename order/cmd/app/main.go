package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/order/application"
	httpinfra "github.com/illia-malachyn/food-delivery/order/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/persistence"
)

func main() {
	log.Println("order microservice started")
	ctx := context.Background()

	connPool, err := pgxpool.New(ctx, databaseURLFromEnv())
	if err != nil {
		log.Fatal(err)
	}

	postgresOrderRepository := persistence.NewPostgresOrderRepository(connPool)
	eventUpcaster := application.NewIntegrationEventUpcaster()
	orderService := application.NewOrderService(postgresOrderRepository, eventUpcaster)
	orderHandler := httpinfra.CreateOrderHandler(orderService)

	http.Handle("/orders", orderHandler)
	if err := http.ListenAndServe(":9876", nil); err != nil {
		log.Fatal(err)
	}
}

func databaseURLFromEnv() string {
	dbUser := getEnvOrDefault("ORDER_DB_USER", "orders_user")
	dbPassword := getEnvOrDefault("ORDER_DB_PASSWORD", "orders_password")
	dbHost := getEnvOrDefault("ORDER_DB_HOST", "localhost")
	dbPort := getEnvOrDefault("ORDER_DB_PORT", "5432")
	dbName := getEnvOrDefault("ORDER_DB_NAME", "orders")
	sslMode := getEnvOrDefault("ORDER_DB_SSLMODE", "disable")

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

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
