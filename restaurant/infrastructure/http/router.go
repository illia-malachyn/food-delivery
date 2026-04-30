package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from restaurant service"))
	})

	return sharedmiddleware.Chain(
		mux,
		sharedmiddleware.Logging(),
		sharedmiddleware.Metrics("restaurant"),
	)
}
