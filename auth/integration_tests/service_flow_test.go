//go:build integration

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/illia-malachyn/food-delivery/auth/internal/auth"
	"github.com/illia-malachyn/food-delivery/auth/internal/config"
	"github.com/illia-malachyn/food-delivery/auth/internal/security"
	"github.com/illia-malachyn/food-delivery/auth/internal/session"
	"github.com/illia-malachyn/food-delivery/auth/internal/user"
)

func TestServiceIntegration_RegisterRefreshLogoutFlow(t *testing.T) {
	ctx := context.Background()

	pgDSN := startPostgresContainer(t, ctx)
	redisAddr := startRedisContainer(t, ctx)

	db := openPostgresPool(t, ctx, pgDSN)
	t.Cleanup(db.Close)
	createUsersTable(t, ctx, db)

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	t.Cleanup(func() { _ = redisClient.Close() })
	require.NoError(t, redisClient.Ping(ctx).Err())

	tokenManager := security.NewJWTManager(config.JWTConfig{
		Issuer:        "auth-integration-test",
		AccessSecret:  "integration-access-secret-very-long",
		RefreshSecret: "integration-refresh-secret-very-long",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    time.Hour,
	})

	refreshStore := session.NewRedisRefreshStore(redisClient)
	usersRepo := user.NewPostgresRepository(db)
	svc := auth.NewService(usersRepo, security.NewBcryptHasher(), tokenManager, refreshStore, time.Hour)

	registered, err := svc.Register(ctx, "Alice@Example.com", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, registered.AccessToken)
	require.NotEmpty(t, registered.RefreshToken)

	firstPrincipal, err := tokenManager.ParseRefresh(registered.RefreshToken)
	require.NoError(t, err)

	exists, err := refreshStore.ExistsForUser(ctx, firstPrincipal.TokenID, firstPrincipal.UserID)
	require.NoError(t, err)
	require.True(t, exists)

	refreshed, err := svc.Refresh(ctx, registered.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, refreshed.RefreshToken)
	assert.NotEqual(t, registered.RefreshToken, refreshed.RefreshToken)

	exists, err = refreshStore.ExistsForUser(ctx, firstPrincipal.TokenID, firstPrincipal.UserID)
	require.NoError(t, err)
	require.False(t, exists)

	secondPrincipal, err := tokenManager.ParseRefresh(refreshed.RefreshToken)
	require.NoError(t, err)

	exists, err = refreshStore.ExistsForUser(ctx, secondPrincipal.TokenID, secondPrincipal.UserID)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, svc.Logout(ctx, refreshed.RefreshToken))

	exists, err = refreshStore.ExistsForUser(ctx, secondPrincipal.TokenID, secondPrincipal.UserID)
	require.NoError(t, err)
	require.False(t, exists)

	_, err = svc.Refresh(ctx, refreshed.RefreshToken)
	require.ErrorIs(t, err, auth.ErrRefreshTokenRevoked)
}

func startPostgresContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.3-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "auth",
				"POSTGRES_USER":     "auth_user",
				"POSTGRES_PASSWORD": "auth_password",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	return fmt.Sprintf("postgres://auth_user:auth_password@%s:%s/auth?sslmode=disable", host, port.Port())
}

func startRedisContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8.4-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)

	return fmt.Sprintf("%s:%s", host, port.Port())
}

func openPostgresPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	deadline := time.Now().Add(30 * time.Second)
	for {
		err = pool.Ping(ctx)
		if err == nil {
			return pool
		}
		if time.Now().After(deadline) {
			pool.Close()
			require.NoError(t, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func createUsersTable(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	require.NoError(t, err)
}
