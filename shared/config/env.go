package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type DBDefaults struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
	SSLMode  string
}

func DatabaseURL(prefix string, defaults DBDefaults) string {
	upperPrefix := strings.ToUpper(strings.TrimSpace(prefix))

	dbUser := GetMany([]string{upperPrefix + "_DB_USER", "DB_USER"}, defaults.User)
	dbPassword := GetMany([]string{upperPrefix + "_DB_PASSWORD", "DB_PASSWORD"}, defaults.Password)
	dbHost := GetMany([]string{upperPrefix + "_DB_HOST", "DB_HOST"}, defaults.Host)
	dbPort := GetMany([]string{upperPrefix + "_DB_PORT", "DB_PORT"}, defaults.Port)
	dbName := GetMany([]string{upperPrefix + "_DB_NAME", "DB_NAME"}, defaults.Name)
	sslMode := GetMany([]string{upperPrefix + "_DB_SSLMODE", "DB_SSL_MODE"}, defaults.SSLMode)

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

func BrokersFromEnv(key string, fallback string) []string {
	raw := GetOrDefault(key, fallback)
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))

	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}

	if len(brokers) == 0 {
		return []string{fallback}
	}

	return brokers
}

func DurationFromEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration for %s=%q, using fallback %s", key, raw, fallback)
		return fallback
	}

	return parsed
}

func IntFromEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid int for %s=%q, using fallback %d", key, raw, fallback)
		return fallback
	}

	return parsed
}

func JWTPublicKeyFromEnv(prefix string) string {
	upperPrefix := strings.ToUpper(strings.TrimSpace(prefix))

	if path := strings.TrimSpace(GetMany([]string{upperPrefix + "_JWT_PUBLIC_KEY_PATH", "JWT_PUBLIC_KEY_PATH"}, "")); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("cannot read JWT public key file %q: %v", path, err)
		}
		return string(content)
	}

	key := strings.TrimSpace(GetMany([]string{upperPrefix + "_JWT_PUBLIC_KEY", "JWT_PUBLIC_KEY"}, ""))
	if key == "" {
		log.Fatalf("JWT public key is not configured; set %s_JWT_PUBLIC_KEY(_PATH) or JWT_PUBLIC_KEY(_PATH)", upperPrefix)
	}
	return key
}

func JWTIssuerFromEnv(prefix string) string {
	upperPrefix := strings.ToUpper(strings.TrimSpace(prefix))
	return GetMany([]string{upperPrefix + "_JWT_ISSUER", "JWT_ISSUER"}, "")
}

func GetMany(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func GetOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
