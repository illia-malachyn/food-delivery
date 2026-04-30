package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func DatabasePoolConfig(prefix string, databaseURL string) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database pool config: %w", err)
	}

	upperPrefix := strings.ToUpper(strings.TrimSpace(prefix))
	poolConfig.MaxConns = int32(IntFromEnvMany([]string{upperPrefix + "_DB_MAX_CONNS", "DB_MAX_CONNS"}, 10))
	poolConfig.MinConns = int32(IntFromEnvMany([]string{upperPrefix + "_DB_MIN_CONNS", "DB_MIN_CONNS"}, 1))
	poolConfig.MaxConnLifetime = DurationFromEnvMany([]string{upperPrefix + "_DB_MAX_CONN_LIFETIME", "DB_MAX_CONN_LIFETIME"}, 30*time.Minute)
	poolConfig.MaxConnIdleTime = DurationFromEnvMany([]string{upperPrefix + "_DB_MAX_CONN_IDLE_TIME", "DB_MAX_CONN_IDLE_TIME"}, 5*time.Minute)
	poolConfig.HealthCheckPeriod = DurationFromEnvMany([]string{upperPrefix + "_DB_HEALTH_CHECK_PERIOD", "DB_HEALTH_CHECK_PERIOD"}, 30*time.Second)
	poolConfig.ConnConfig.ConnectTimeout = DurationFromEnvMany([]string{upperPrefix + "_DB_CONNECT_TIMEOUT", "DB_CONNECT_TIMEOUT"}, 5*time.Second)

	return poolConfig, nil
}
