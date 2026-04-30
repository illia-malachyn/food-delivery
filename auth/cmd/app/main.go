package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/illia-malachyn/food-delivery/auth/internal/auth"
	"github.com/illia-malachyn/food-delivery/auth/internal/config"
	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi"
	"github.com/illia-malachyn/food-delivery/auth/internal/security"
	"github.com/illia-malachyn/food-delivery/auth/internal/session"
	"github.com/illia-malachyn/food-delivery/auth/internal/user"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dbPoolConfig, err := cfg.DatabasePoolConfig()
	if err != nil {
		log.Fatalf("cannot configure postgres pool: %v", err)
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, dbPoolConfig)
	if err != nil {
		log.Fatalf("cannot connect to postgres: %v", err)
	}
	defer dbPool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot connect to redis: %v", err)
	}
	defer redisClient.Close()

	userRepository := user.NewPostgresRepository(dbPool)
	passwordHasher := security.NewBcryptHasher()
	tokenManager, err := security.NewJWTManager(cfg.JWT)
	if err != nil {
		log.Fatalf("cannot initialize JWT manager: %v", err)
	}
	refreshStore := session.NewRedisRefreshStore(redisClient)
	authService := auth.NewService(userRepository, passwordHasher, tokenManager, refreshStore, cfg.JWT.RefreshTTL)
	handler := httpapi.NewHandler(authService, cfg.Cookie, cfg.JWT.RefreshTTL, cfg.HTTP.RequestTimeout)
	router := httpapi.NewRouter(handler, tokenManager)

	server := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("auth server shutdown failed: %v", err)
		}
	}()

	log.Printf("auth microservice started on :%s", cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("auth server stopped: %v", err)
	}
}
