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

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure"
	httpinfra "github.com/illia-malachyn/food-delivery/payment/infrastructure/http"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/persistence"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	sharedjwt "github.com/illia-malachyn/food-delivery/shared/security/jwt"
)

func main() {
	log.Println("payment microservice started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connPool, err := pgxpool.New(ctx, sharedconfig.DatabaseURL("payment", sharedconfig.DBDefaults{
		User:     "payments_user",
		Password: "payments_password",
		Host:     "localhost",
		Port:     "5432",
		Name:     "payments",
		SSLMode:  "disable",
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer connPool.Close()

	paymentRepository := persistence.NewPostgresPaymentRepository(connPool)
	paymentService := application.NewPaymentService(paymentRepository)

	orderEventsConsumer := infrastructure.NewOrderEventsConsumer(
		sharedconfig.BrokersFromEnv("KAFKA_BROKERS", "localhost:9092"),
		sharedconfig.GetOrDefault("KAFKA_TOPIC_ORDER_EVENTS", "order.events"),
		sharedconfig.GetMany([]string{"PAYMENT_KAFKA_GROUP_ID", "KAFKA_GROUP_ID"}, "payment-service"),
		paymentService,
		paymentRepository,
		int64(sharedconfig.IntFromEnv("PAYMENT_DEFAULT_AMOUNT", 1000)),
		sharedconfig.GetMany([]string{"PAYMENT_DEFAULT_CURRENCY"}, "USD"),
	)
	defer orderEventsConsumer.Close()

	jwtVerifier, err := sharedjwt.NewVerifier(
		sharedconfig.JWTPublicKeyFromEnv("payment"),
		sharedconfig.JWTIssuerFromEnv("payment"),
	)
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	router := httpinfra.NewRouter(paymentService, sharedmiddleware.RequireJWT(jwtVerifier))

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

	go orderEventsConsumer.Run(ctx)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
