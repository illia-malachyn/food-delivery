package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Middleware func(http.Handler) http.Handler

type jwtVerifier interface {
	VerifyAccessToken(token string) error
}

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests handled by the service.",
	},
	[]string{"service", "method", "path", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

func Metrics(service string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			httpRequestsTotal.WithLabelValues(
				service,
				r.Method,
				normalizePath(r),
				strconv.Itoa(rec.status),
			).Inc()
		})
	}
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

func normalizePath(r *http.Request) string {
	path := r.Pattern
	if path == "" {
		path = r.URL.Path
	}
	if strings.Contains(path, " ") {
		parts := strings.SplitN(path, " ", 2)
		path = parts[1]
	}
	return path
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
