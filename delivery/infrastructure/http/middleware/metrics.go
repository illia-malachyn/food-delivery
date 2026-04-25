package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests handled by the delivery service.",
	},
	[]string{"service", "method", "path", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
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
