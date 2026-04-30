package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/infrastructure/http/middleware"
)

func NewRouter(paymentService *application.PaymentService) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from payment service"))
	})
	mux.Handle("POST /payments", CreatePaymentHandler(paymentService))
	mux.Handle("POST /payments/{id}/pay", MarkPaidHandler(paymentService))
	mux.Handle("POST /payments/{id}/fail", MarkFailedHandler(paymentService))
	mux.Handle("POST /payments/{id}/refund", RefundPaymentHandler(paymentService))

	return middleware.Chain(
		mux,
		middleware.Logging(),
		middleware.Metrics("payment"),
	)
}
