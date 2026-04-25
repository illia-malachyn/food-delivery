//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthGatewayFlow(t *testing.T) {
	baseURL := envOrDefault("E2E_BASE_URL", defaultBaseURL)
	client := newHTTPClient(t)

	waitForHTTP(t, baseURL+"/healthz")
	waitForHTTP(t, baseURL+"/auth/healthz")

	email := uniqueEmail("e2e-auth")
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

	meHeaders := map[string]string{"Authorization": "Bearer " + loginToken.AccessToken}
	meRes := doJSON(t, client, http.MethodGet, baseURL+"/auth/me", nil, meHeaders)
	defer meRes.Body.Close()
	requireStatus(t, meRes, http.StatusOK)

	var principal principalResponse
	decodeJSON(t, meRes.Body, &principal)
	assert.Equal(t, email, principal.Email)

	refreshRes := doJSON(t, client, http.MethodPost, baseURL+"/auth/refresh", map[string]any{}, nil)
	defer refreshRes.Body.Close()
	requireStatus(t, refreshRes, http.StatusOK)

	logoutRes := doJSON(t, client, http.MethodPost, baseURL+"/auth/logout", map[string]any{}, nil)
	defer logoutRes.Body.Close()
	requireStatus(t, logoutRes, http.StatusNoContent)

	refreshAfterLogoutRes := doJSON(t, client, http.MethodPost, baseURL+"/auth/refresh", map[string]any{}, nil)
	defer refreshAfterLogoutRes.Body.Close()
	assertStatus(t, refreshAfterLogoutRes, http.StatusUnauthorized)
}
