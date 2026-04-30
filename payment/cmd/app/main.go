package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/payment/application"
	httpinfra "github.com/illia-malachyn/food-delivery/payment/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/http/middleware"
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

	jwtVerifier, err := httpinfra.NewJWTVerifier(jwtPublicKeyFromEnv(), jwtIssuerFromEnv())
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	router := httpinfra.NewRouter(paymentService, middleware.RequireJWT(jwtVerifier))

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

func jwtPublicKeyFromEnv() string {
	if path := strings.TrimSpace(getEnvOrDefaultMany([]string{"PAYMENT_JWT_PUBLIC_KEY_PATH", "JWT_PUBLIC_KEY_PATH"}, "")); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("cannot read JWT public key file %q: %v", path, err)
		}
		return string(content)
	}

	key := strings.TrimSpace(getEnvOrDefaultMany([]string{"PAYMENT_JWT_PUBLIC_KEY", "JWT_PUBLIC_KEY"}, ""))
	if key == "" {
		log.Fatal("JWT public key is not configured; set PAYMENT_JWT_PUBLIC_KEY(_PATH) or JWT_PUBLIC_KEY(_PATH)")
	}
	return key
}

func jwtIssuerFromEnv() string {
	return getEnvOrDefaultMany([]string{"PAYMENT_JWT_ISSUER", "JWT_ISSUER"}, "")
}
