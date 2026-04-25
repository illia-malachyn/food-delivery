package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illia-malachyn/food-delivery/auth/internal/security"
)

type stubParser struct{}

func (stubParser) ParseAccess(token string) (security.AccessPrincipal, error) {
	if token == "valid-token" {
		return security.AccessPrincipal{UserID: 11, Email: "test@example.com"}, nil
	}
	return security.AccessPrincipal{}, security.ErrInvalidAccessToken
}

func TestRequireAuthAllowsRequestWithValidToken(t *testing.T) {
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("principal missing in context")
		}
		if principal.UserID != 11 {
			t.Fatalf("unexpected user id: %d", principal.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}), RequireAuth(stubParser{}))

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestRequireAuthRejectsMissingHeader(t *testing.T) {
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), RequireAuth(stubParser{}))

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
