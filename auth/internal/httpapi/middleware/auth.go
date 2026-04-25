package middleware

import (
	"context"
	"strings"

	"net/http"

	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/httputil"
	"github.com/illia-malachyn/food-delivery/auth/internal/security"
)

type accessParser interface {
	ParseAccess(token string) (security.AccessPrincipal, error)
}

type principalContextKey struct{}

func RequireAuth(parser accessParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			principal, err := parser.ParseAccess(token)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid access token")
				return
			}

			ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PrincipalFromContext(ctx context.Context) (security.AccessPrincipal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(security.AccessPrincipal)
	return principal, ok
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
