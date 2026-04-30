package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubJWTVerifier struct {
	err error
}

func (s stubJWTVerifier) VerifyAccessToken(token string) error {
	if token == "valid-token" {
		return nil
	}
	if s.err != nil {
		return s.err
	}
	return errors.New("invalid")
}

func TestRequireJWT_AllowsRequestWithValidBearerToken(t *testing.T) {
	t.Parallel()

	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireJWT_RejectsMissingHeader(t *testing.T) {
	t.Parallel()

	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireJWT_RejectsInvalidToken(t *testing.T) {
	t.Parallel()

	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), RequireJWT(stubJWTVerifier{}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	req.Header.Set("Authorization", "Bearer broken")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
