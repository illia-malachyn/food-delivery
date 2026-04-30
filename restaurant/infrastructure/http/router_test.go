package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
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

	router := NewRouter(sharedmiddleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRouter_MetricsDoesNotRequireJWT(t *testing.T) {
	t.Parallel()

	router := NewRouter(sharedmiddleware.RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
