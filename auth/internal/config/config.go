package config

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL string
	Database    DatabaseConfig
	HTTP        HTTPConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Cookie      CookieConfig
}

type DatabaseConfig struct {
	MaxConns          int32
	MinConns          int32
	ConnectTimeout    time.Duration
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type HTTPConfig struct {
	Port           string
	RequestTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type JWTConfig struct {
	Issuer        string
	PrivateKeyPEM string
	PublicKeyPEM  string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

func Load() Config {
	cfg := Config{
		DatabaseURL: databaseURLFromEnv(),
		Database: DatabaseConfig{
			MaxConns:          int32(intFromEnvMany([]string{"AUTH_DB_MAX_CONNS", "DB_MAX_CONNS"}, 10)),
			MinConns:          int32(intFromEnvMany([]string{"AUTH_DB_MIN_CONNS", "DB_MIN_CONNS"}, 1)),
			ConnectTimeout:    durationFromEnvMany([]string{"AUTH_DB_CONNECT_TIMEOUT", "DB_CONNECT_TIMEOUT"}, 5*time.Second),
			MaxConnLifetime:   durationFromEnvMany([]string{"AUTH_DB_MAX_CONN_LIFETIME", "DB_MAX_CONN_LIFETIME"}, 30*time.Minute),
			MaxConnIdleTime:   durationFromEnvMany([]string{"AUTH_DB_MAX_CONN_IDLE_TIME", "DB_MAX_CONN_IDLE_TIME"}, 5*time.Minute),
			HealthCheckPeriod: durationFromEnvMany([]string{"AUTH_DB_HEALTH_CHECK_PERIOD", "DB_HEALTH_CHECK_PERIOD"}, 30*time.Second),
		},
		HTTP: HTTPConfig{
			Port:           getEnvOrDefault("HTTP_PORT", "8080"),
			RequestTimeout: durationFromEnv("HTTP_REQUEST_TIMEOUT", 3*time.Second),
			ReadTimeout:    durationFromEnv("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:   durationFromEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:    durationFromEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Redis: RedisConfig{
			Address:  getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       intFromEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Issuer:        getEnvOrDefault("JWT_ISSUER", "food-delivery-auth"),
			PrivateKeyPEM: jwtKeyFromEnv("JWT_PRIVATE_KEY_PATH", "JWT_PRIVATE_KEY"),
			PublicKeyPEM:  jwtKeyFromEnv("JWT_PUBLIC_KEY_PATH", "JWT_PUBLIC_KEY"),
			AccessTTL:     durationFromEnv("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    durationFromEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Cookie: CookieConfig{
			Name:     getEnvOrDefault("REFRESH_COOKIE_NAME", "refresh_token"),
			Domain:   strings.TrimSpace(os.Getenv("REFRESH_COOKIE_DOMAIN")),
			Path:     getEnvOrDefault("REFRESH_COOKIE_PATH", "/"),
			Secure:   boolFromEnv("REFRESH_COOKIE_SECURE", false),
			HTTPOnly: boolFromEnv("REFRESH_COOKIE_HTTP_ONLY", true),
			SameSite: sameSiteFromEnv("REFRESH_COOKIE_SAME_SITE", http.SameSiteLaxMode),
		},
	}

	return cfg
}

func (cfg Config) DatabasePoolConfig() (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database pool config: %w", err)
	}

	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.ConnConfig.ConnectTimeout = cfg.Database.ConnectTimeout
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Database.HealthCheckPeriod

	return poolConfig, nil
}

func databaseURLFromEnv() string {
	dbUser := getEnvOrDefaultMany([]string{"AUTH_DB_USER", "DB_USER"}, "auth_user")
	dbPassword := getEnvOrDefaultMany([]string{"AUTH_DB_PASSWORD", "DB_PASSWORD"}, "auth_password")
	dbHost := getEnvOrDefaultMany([]string{"AUTH_DB_HOST", "DB_HOST"}, "localhost")
	dbPort := getEnvOrDefaultMany([]string{"AUTH_DB_PORT", "DB_PORT"}, "5432")
	dbName := getEnvOrDefaultMany([]string{"AUTH_DB_NAME", "DB_NAME"}, "auth")
	sslMode := getEnvOrDefaultMany([]string{"AUTH_DB_SSLMODE", "DB_SSL_MODE"}, "disable")

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

func sameSiteFromEnv(key string, fallback http.SameSite) http.SameSite {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax", "":
		return http.SameSiteLaxMode
	default:
		log.Printf("invalid same-site for %s=%q, using fallback", key, value)
		return fallback
	}
}

func boolFromEnv(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("invalid bool for %s=%q, using fallback %t", key, raw, fallback)
		return fallback
	}

	return parsed
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	return durationFromEnvMany([]string{key}, fallback)
}

func durationFromEnvMany(keys []string, fallback time.Duration) time.Duration {
	raw := ""
	usedKey := ""
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			raw = value
			usedKey = key
			break
		}
	}
	if raw == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration for %s=%q, using fallback %s", usedKey, raw, fallback)
		return fallback
	}

	return parsed
}

func intFromEnv(key string, fallback int) int {
	return intFromEnvMany([]string{key}, fallback)
}

func intFromEnvMany(keys []string, fallback int) int {
	raw := ""
	usedKey := ""
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			raw = value
			usedKey = key
			break
		}
	}
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid int for %s=%q, using fallback %d", usedKey, raw, fallback)
		return fallback
	}

	return parsed
}

func getEnvOrDefaultMany(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func jwtKeyFromEnv(pathKey string, inlineKey string) string {
	if path := strings.TrimSpace(os.Getenv(pathKey)); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("cannot read JWT key file %s=%q: %v", pathKey, path, err)
		}
		return string(content)
	}

	if inline := strings.TrimSpace(os.Getenv(inlineKey)); inline != "" {
		return strings.ReplaceAll(inline, `\n`, "\n")
	}

	log.Fatalf("JWT key is not configured; set %s or %s", pathKey, inlineKey)
	return ""
}
