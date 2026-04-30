package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

type jwtVerifier interface {
	VerifyAccessToken(token string) error
}

func RequireJWT(verifier jwtVerifier) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w, "missing or invalid authorization header")
				return
			}

			if err := verifier.VerifyAccessToken(token); err != nil {
				writeUnauthorized(w, "invalid access token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(rawHeader string) (string, bool) {
	value := strings.TrimSpace(rawHeader)
	if value == "" {
		return "", false
	}

	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
