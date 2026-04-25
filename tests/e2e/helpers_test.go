//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultBaseURL  = "http://localhost:8080"
	defaultOrderDSN = "postgres://orders_user:orders_password@localhost:5432/orders?sslmode=disable"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type principalResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
}

func newHTTPClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
	}
}

func openOrderDB(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	return pool
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("service is not reachable: %s", url)
}

func waitForOrderByUserAndItem(t *testing.T, db *pgxpool.Pool, userID, itemID string) (string, string) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var id, status string
		err := db.QueryRow(
			ctx,
			`SELECT id, status
			 FROM orders
			 WHERE user_id = $1 AND item_id = $2
			 ORDER BY created_at DESC
			 LIMIT 1`,
			userID,
			itemID,
		).Scan(&id, &status)
		cancel()

		if err == nil {
			return id, status
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			require.NoError(t, err)
		}

		time.Sleep(400 * time.Millisecond)
	}

	t.Fatalf("order not found for user_id=%s item_id=%s", userID, itemID)
	return "", ""
}

func waitForOrderStatus(t *testing.T, db *pgxpool.Pool, orderID, expectedStatus string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var status string
		err := db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
		cancel()

		if err == nil && status == expectedStatus {
			return
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			require.NoError(t, err)
		}

		time.Sleep(350 * time.Millisecond)
	}

	t.Fatalf("order %s did not reach status %q", orderID, expectedStatus)
}

func waitForOutboxEventCount(t *testing.T, db *pgxpool.Pool, orderID, eventName string, eventVersion int, reason string) int {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		var (
			count int
			err   error
		)

		if reason == "" {
			err = db.QueryRow(
				ctx,
				`SELECT COUNT(*)
				 FROM outbox
				 WHERE aggregate_type = 'order'
				   AND aggregate_id = $1
				   AND event_name = $2
				   AND event_version = $3`,
				orderID,
				eventName,
				eventVersion,
			).Scan(&count)
		} else {
			err = db.QueryRow(
				ctx,
				`SELECT COUNT(*)
				 FROM outbox
				 WHERE aggregate_type = 'order'
				   AND aggregate_id = $1
				   AND event_name = $2
				   AND event_version = $3
				   AND payload->>'reason' = $4`,
				orderID,
				eventName,
				eventVersion,
				reason,
			).Scan(&count)
		}
		cancel()

		require.NoError(t, err)
		if count > 0 {
			return count
		}

		time.Sleep(400 * time.Millisecond)
	}

	return 0
}

func doJSON(t *testing.T, client *http.Client, method, url string, payload any, headers map[string]string) *http.Response {
	t.Helper()

	var body io.Reader = http.NoBody
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func requireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		require.Equalf(t, expected, resp.StatusCode, "body=%s", readBody(resp.Body))
	}
}

func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		assert.Equalf(t, expected, resp.StatusCode, "body=%s", readBody(resp.Body))
	}
}

func decodeJSON(t *testing.T, r io.Reader, out any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r).Decode(out))
}

func readBody(r io.Reader) string {
	b, err := io.ReadAll(r)
	if err != nil {
		return "<failed to read body>"
	}
	return strings.TrimSpace(string(b))
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
