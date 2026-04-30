package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/illia-malachyn/food-delivery/payment/application"
	sharedmiddleware "github.com/illia-malachyn/food-delivery/shared/http/middleware"
)

func NewRouter(paymentService *application.PaymentService, requireAuth sharedmiddleware.Middleware) http.Handler {
	if requireAuth == nil {
		requireAuth = func(next http.Handler) http.Handler { return next }
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from payment service"))
	})
	mux.Handle("POST /payments", requireAuth(CreatePaymentHandler(paymentService)))
	mux.Handle("POST /payments/{id}/pay", requireAuth(MarkPaidHandler(paymentService)))
	mux.Handle("POST /payments/{id}/fail", requireAuth(MarkFailedHandler(paymentService)))
	mux.Handle("POST /payments/{id}/refund", requireAuth(RefundPaymentHandler(paymentService)))

	return sharedmiddleware.Chain(
		mux,
		sharedmiddleware.Logging(),
		sharedmiddleware.Metrics("payment"),
	)
}
