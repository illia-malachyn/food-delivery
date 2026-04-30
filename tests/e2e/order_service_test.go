//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderGatewayFlowWithOutboxAssertions(t *testing.T) {
	baseURL := envOrDefault("E2E_BASE_URL", defaultBaseURL)
	orderDSN := envOrDefault("E2E_ORDER_DB_DSN", defaultOrderDSN)

	waitForHTTP(t, baseURL+"/healthz")
	waitForHTTP(t, baseURL+"/auth/healthz")
	waitForHTTP(t, baseURL+"/orders/healthz")

	db := openOrderDB(t, orderDSN)
	defer db.Close()

	client := newHTTPClient(t)
	email := uniqueEmail("e2e-order")
	password := "password123"

	register := map[string]string{"email": email, "password": password}
	registerRes := doJSON(t, client, http.MethodPost, baseURL+"/auth/register", register, nil)
	defer registerRes.Body.Close()
	requireStatus(t, registerRes, http.StatusCreated)

	login := map[string]string{"email": email, "password": password}
	loginRes := doJSON(t, client, http.MethodPost, baseURL+"/auth/login", login, nil)
	defer loginRes.Body.Close()
	requireStatus(t, loginRes, http.StatusOK)

	var loginToken tokenResponse
	decodeJSON(t, loginRes.Body, &loginToken)
	require.NotEmpty(t, loginToken.AccessToken)

	authHeaders := map[string]string{
		"Authorization": "Bearer " + loginToken.AccessToken,
	}

	userID := uniqueID("u-e2e")
	itemID := uniqueID("pizza")

	createPayload := map[string]any{
		"user_id":  userID,
		"item_id":  itemID,
		"quantity": 2,
	}
	createRes := doJSON(t, client, http.MethodPost, baseURL+"/orders", createPayload, authHeaders)
	defer createRes.Body.Close()
	requireStatus(t, createRes, http.StatusOK)

	orderID, status := waitForOrderByUserAndItem(t, db, userID, itemID)
	require.NotEmpty(t, orderID)
	assert.Equal(t, "placed", status)

	count := waitForOutboxEventCount(t, db, orderID, "OrderPlaced", 2, "")
	require.GreaterOrEqual(t, count, 1)

	cancelPayload := map[string]any{"reason": "customer request"}
	cancelURL := fmt.Sprintf("%s/orders/%s/cancel", baseURL, orderID)
	cancelRes := doJSON(t, client, http.MethodPost, cancelURL, cancelPayload, authHeaders)
	defer cancelRes.Body.Close()
	requireStatus(t, cancelRes, http.StatusOK)

	waitForOrderStatus(t, db, orderID, "cancelled")

	count = waitForOutboxEventCount(t, db, orderID, "OrderCancelled", 1, "customer request")
	require.GreaterOrEqual(t, count, 1)
}
