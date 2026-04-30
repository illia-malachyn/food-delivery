package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/payment/application"
	httpinfra "github.com/illia-malachyn/food-delivery/payment/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/persistence"
)

func main() {
	log.Println("payment microservice started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connPool, err := pgxpool.New(ctx, databaseURLFromEnv())
	if err != nil {
		log.Fatal(err)
	}
	defer connPool.Close()

	paymentRepository := persistence.NewPostgresPaymentRepository(connPool)
	paymentService := application.NewPaymentService(paymentRepository)
	router := httpinfra.NewRouter(paymentService)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("payment server shutdown failed: %v", shutdownErr)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func databaseURLFromEnv() string {
	dbUser := getEnvOrDefaultMany([]string{"PAYMENT_DB_USER", "DB_USER"}, "payments_user")
	dbPassword := getEnvOrDefaultMany([]string{"PAYMENT_DB_PASSWORD", "DB_PASSWORD"}, "payments_password")
	dbHost := getEnvOrDefaultMany([]string{"PAYMENT_DB_HOST", "DB_HOST"}, "localhost")
	dbPort := getEnvOrDefaultMany([]string{"PAYMENT_DB_PORT", "DB_PORT"}, "5432")
	dbName := getEnvOrDefaultMany([]string{"PAYMENT_DB_NAME", "DB_NAME"}, "payments")
	sslMode := getEnvOrDefaultMany([]string{"PAYMENT_DB_SSLMODE", "DB_SSL_MODE"}, "disable")

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

func getEnvOrDefaultMany(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}
