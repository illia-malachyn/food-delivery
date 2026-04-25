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

	db := openOrderDB(t, orderDSN)
	defer db.Close()

	userID := uniqueID("u-e2e")
	itemID := uniqueID("pizza")

	createPayload := map[string]any{
		"user_id":  userID,
		"item_id":  itemID,
		"quantity": 2,
	}
	createRes := doJSON(t, http.DefaultClient, http.MethodPost, baseURL+"/orders", createPayload, nil)
	defer createRes.Body.Close()
	requireStatus(t, createRes, http.StatusOK)

	orderID, status := waitForOrderByUserAndItem(t, db, userID, itemID)
	require.NotEmpty(t, orderID)
	assert.Equal(t, "placed", status)

	count := waitForOutboxEventCount(t, db, orderID, "OrderPlaced", 2, "")
	require.GreaterOrEqual(t, count, 1)

	cancelPayload := map[string]any{"reason": "customer request"}
	cancelURL := fmt.Sprintf("%s/orders/%s/cancel", baseURL, orderID)
	cancelRes := doJSON(t, http.DefaultClient, http.MethodPost, cancelURL, cancelPayload, nil)
	defer cancelRes.Body.Close()
	requireStatus(t, cancelRes, http.StatusOK)

	waitForOrderStatus(t, db, orderID, "cancelled")

	count = waitForOutboxEventCount(t, db, orderID, "OrderCancelled", 1, "customer request")
	require.GreaterOrEqual(t, count, 1)
}
