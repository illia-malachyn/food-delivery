package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabasePoolConfigUsesServiceOverrides(t *testing.T) {
	t.Setenv("ORDER_DB_MAX_CONNS", "7")
	t.Setenv("ORDER_DB_MIN_CONNS", "2")
	t.Setenv("ORDER_DB_CONNECT_TIMEOUT", "250ms")
	t.Setenv("DB_MAX_CONNS", "20")

	cfg, err := DatabasePoolConfig("order", "postgres://user:pass@localhost:5432/orders?sslmode=disable")
	require.NoError(t, err)

	assert.Equal(t, int32(7), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	assert.Equal(t, 250*time.Millisecond, cfg.ConnConfig.ConnectTimeout)
}
