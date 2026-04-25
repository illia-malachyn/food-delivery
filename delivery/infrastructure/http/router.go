package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/delivery/infrastructure/http/middleware"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from delivery service"))
	})

	return middleware.Chain(
		mux,
		middleware.Logging(),
		middleware.Metrics("delivery"),
	)
}
