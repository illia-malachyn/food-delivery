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
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/provider"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	"github.com/illia-malachyn/food-delivery/shared/resilience"
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

	paymentProvider := application.NewNoopPaymentProvider()
	if paymentProviderURL := sharedconfig.GetOrDefault("PAYMENT_PROVIDER_URL", ""); paymentProviderURL != "" {
		paymentProvider = provider.NewHTTPPaymentProvider(paymentProviderURL, &http.Client{
			Transport: resilience.NewCircuitBreakerRoundTripper(
				resilience.NewRetryRoundTripper(http.DefaultTransport, resilience.RetryPolicy{
					MaxAttempts:    sharedconfig.IntFromEnv("PAYMENT_PROVIDER_HTTP_RETRY_MAX_ATTEMPTS", 3),
					InitialBackoff: sharedconfig.DurationFromEnv("PAYMENT_PROVIDER_HTTP_RETRY_INITIAL_BACKOFF", 100*time.Millisecond),
					MaxBackoff:     sharedconfig.DurationFromEnv("PAYMENT_PROVIDER_HTTP_RETRY_MAX_BACKOFF", 2*time.Second),
					Jitter:         sharedconfig.FloatFromEnv("PAYMENT_PROVIDER_HTTP_RETRY_JITTER", 0.2),
				}),
				resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
					FailureThreshold: sharedconfig.IntFromEnv("PAYMENT_PROVIDER_CIRCUIT_FAILURE_THRESHOLD", 5),
					OpenTimeout:      sharedconfig.DurationFromEnv("PAYMENT_PROVIDER_CIRCUIT_OPEN_TIMEOUT", 30*time.Second),
				}),
			),
			Timeout: sharedconfig.DurationFromEnv("PAYMENT_PROVIDER_HTTP_TIMEOUT", 2*time.Second),
		})
	}

	paymentRepository := persistence.NewPostgresPaymentRepository(connPool)
	paymentService := application.NewPaymentService(paymentRepository, paymentProvider)

	orderEventsConsumer := infrastructure.NewOrderEventsConsumer(
		sharedconfig.BrokersFromEnv("KAFKA_BROKERS", "localhost:9092"),
		sharedconfig.GetOrDefault("KAFKA_TOPIC_ORDER_EVENTS", "order.events"),
		sharedconfig.GetMany([]string{"PAYMENT_KAFKA_GROUP_ID", "KAFKA_GROUP_ID"}, "payment-service"),
		paymentService,
		paymentRepository,
		int64(sharedconfig.IntFromEnv("PAYMENT_DEFAULT_AMOUNT", 1000)),
		sharedconfig.GetMany([]string{"PAYMENT_DEFAULT_CURRENCY"}, "USD"),
		infrastructure.WithRetryPolicy(resilience.RetryPolicy{
			MaxAttempts:    sharedconfig.IntFromEnv("PAYMENT_CONSUMER_RETRY_MAX_ATTEMPTS", 3),
			InitialBackoff: sharedconfig.DurationFromEnv("PAYMENT_CONSUMER_RETRY_INITIAL_BACKOFF", 100*time.Millisecond),
			MaxBackoff:     sharedconfig.DurationFromEnv("PAYMENT_CONSUMER_RETRY_MAX_BACKOFF", 2*time.Second),
			Jitter:         sharedconfig.FloatFromEnv("PAYMENT_CONSUMER_RETRY_JITTER", 0.2),
		}),
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
