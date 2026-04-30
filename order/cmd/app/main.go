package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/infrastructure"
	httpinfra "github.com/illia-malachyn/food-delivery/order/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/persistence"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	"github.com/illia-malachyn/food-delivery/shared/resilience"
	sharedjwt "github.com/illia-malachyn/food-delivery/shared/security/jwt"
)

func main() {
	log.Println("order microservice started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := sharedconfig.DatabaseURL("order", sharedconfig.DBDefaults{
		User:     "orders_user",
		Password: "orders_password",
		Host:     "localhost",
		Port:     "5432",
		Name:     "orders",
		SSLMode:  "disable",
	})
	poolConfig, err := sharedconfig.DatabasePoolConfig("order", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer connPool.Close()

	kafkaPublisher, err := infrastructure.NewKafkaOutboxPublisher(
		sharedconfig.BrokersFromEnv("KAFKA_BROKERS", "localhost:9092"),
		sharedconfig.GetOrDefault("KAFKA_TOPIC_ORDER_EVENTS", "order.events"),
	)
	if err != nil {
		log.Fatalf("cannot initialize kafka publisher: %v", err)
	}
	defer kafkaPublisher.Close()

	outboxRelay := infrastructure.NewOutboxRelay(
		connPool,
		kafkaPublisher,
		sharedconfig.IntFromEnv("OUTBOX_BATCH_SIZE", 100),
		sharedconfig.DurationFromEnv("OUTBOX_POLL_INTERVAL", 2*time.Second),
	)
	go outboxRelay.Run(ctx)

	postgresOrderRepository := persistence.NewPostgresOrderRepository(connPool)
	eventUpcaster := application.NewIntegrationEventUpcaster()
	orderService := application.NewOrderService(postgresOrderRepository, eventUpcaster)

	jwtVerifier, err := sharedjwt.NewVerifier(
		sharedconfig.JWTPublicKeyFromEnv("order"),
		sharedconfig.JWTIssuerFromEnv("order"),
	)
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	router := httpinfra.NewRouter(orderService, sharedmiddleware.RequireJWT(jwtVerifier))

	server := &http.Server{
		Addr:         ":8080",
		Handler:      resilience.NewTimeoutHandler(router, sharedconfig.DurationFromEnv("HTTP_REQUEST_TIMEOUT", 3*time.Second)),
		ReadTimeout:  sharedconfig.DurationFromEnv("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: sharedconfig.DurationFromEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  sharedconfig.DurationFromEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
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
