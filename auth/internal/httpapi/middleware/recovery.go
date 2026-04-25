package middleware

import (
	"log"
	"net/http"

	"github.com/illia-malachyn/food-delivery/auth/internal/httpapi/httputil"
)

func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Printf("panic recovered: %v", recovered)
					httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
