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

	httpinfra "github.com/illia-malachyn/food-delivery/delivery/infrastructure/http"
	sharedconfig "github.com/illia-malachyn/food-delivery/shared/config"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
	"github.com/illia-malachyn/food-delivery/shared/resilience"
	sharedjwt "github.com/illia-malachyn/food-delivery/shared/security/jwt"
)

func main() {
	log.Println("delivery microservice started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := sharedconfig.DatabaseURL("delivery", sharedconfig.DBDefaults{
		User:     "deliveries_user",
		Password: "deliveries_password",
		Host:     "localhost",
		Port:     "5432",
		Name:     "deliveries",
		SSLMode:  "disable",
	})
	poolConfig, err := sharedconfig.DatabasePoolConfig("delivery", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer connPool.Close()

	jwtVerifier, err := sharedjwt.NewVerifier(
		sharedconfig.JWTPublicKeyFromEnv("delivery"),
		sharedconfig.JWTIssuerFromEnv("delivery"),
	)
	if err != nil {
		log.Fatalf("cannot initialize JWT verifier: %v", err)
	}

	server := &http.Server{
		Addr:         ":8080",
		Handler:      resilience.NewTimeoutHandler(httpinfra.NewRouter(sharedmiddleware.RequireJWT(jwtVerifier)), sharedconfig.DurationFromEnv("HTTP_REQUEST_TIMEOUT", 3*time.Second)),
		ReadTimeout:  sharedconfig.DurationFromEnv("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: sharedconfig.DurationFromEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  sharedconfig.DurationFromEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("delivery server shutdown failed: %v", shutdownErr)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("delivery server stopped: %v", err)
	}
}
