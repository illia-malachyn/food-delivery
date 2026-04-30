package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/application/mocks"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/http/middleware"
	"github.com/stretchr/testify/require"
)

type stubJWTVerifier struct{}

func (stubJWTVerifier) VerifyAccessToken(token string) error {
	if token == "valid-token" {
		return nil
	}
	return errors.New("invalid token")
}

func TestRouter_ProtectedEndpointRequiresJWT(t *testing.T) {
	t.Parallel()

	repository := mocks.NewOrderRepository(t)
	upcaster := mocks.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	router := NewRouter(service, middleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"user_id":"user-1","item_id":"item-1","quantity":1}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_PublicEndpointDoesNotRequireJWT(t *testing.T) {
	t.Parallel()

	repository := mocks.NewOrderRepository(t)
	upcaster := mocks.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	router := NewRouter(service, middleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
