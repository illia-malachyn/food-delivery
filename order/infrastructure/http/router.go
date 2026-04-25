package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/infrastructure/http/middleware"
)

func NewRouter(orderService *application.OrderService) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.Handle("/orders", CreateOrderHandler(orderService))
	mux.Handle("/orders/{id}/confirm", ConfirmOrderHandler(orderService))
	mux.Handle("/orders/{id}/cancel", CancelOrderHandler(orderService))

	return middleware.Chain(
		mux,
		middleware.Logging(),
		middleware.Metrics("order"),
	)
}
