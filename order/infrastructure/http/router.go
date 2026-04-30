package http

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/order/application"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
)

func NewRouter(orderService *application.OrderService, requireAuth sharedmiddleware.Middleware) http.Handler {
	if requireAuth == nil {
		requireAuth = func(next http.Handler) http.Handler { return next }
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("POST /orders", requireAuth(CreateOrderHandler(orderService)))
	mux.Handle("POST /orders/{id}/confirm", requireAuth(ConfirmOrderHandler(orderService)))
	mux.Handle("POST /orders/{id}/cancel", requireAuth(CancelOrderHandler(orderService)))

	return sharedmiddleware.Chain(
		mux,
		sharedmiddleware.Logging(),
		sharedmiddleware.Metrics("order"),
	)
}
