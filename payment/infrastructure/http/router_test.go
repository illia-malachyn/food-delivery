package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/http/middleware"
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

	service := application.NewPaymentService(repositoryStub{
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	router := NewRouter(service, middleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{"order_id":"order-1","amount":1200,"currency":"USD"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_PublicEndpointDoesNotRequireJWT(t *testing.T) {
	t.Parallel()

	service := application.NewPaymentService(repositoryStub{
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	router := NewRouter(service, middleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
