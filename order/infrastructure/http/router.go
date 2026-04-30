package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/http/middleware"
)

func NewRouter(orderService *application.OrderService, requireAuth middleware.Middleware) http.Handler {
	if requireAuth == nil {
		requireAuth = func(next http.Handler) http.Handler { return next }
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("POST /orders", requireAuth(CreateOrderHandler(orderService)))
	mux.Handle("POST /orders/{id}/confirm", requireAuth(ConfirmOrderHandler(orderService)))
	mux.Handle("POST /orders/{id}/cancel", requireAuth(CancelOrderHandler(orderService)))

	return middleware.Chain(
		mux,
		middleware.Logging(),
		middleware.Metrics("order"),
	)
}
