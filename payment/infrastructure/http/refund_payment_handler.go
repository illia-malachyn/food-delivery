package http

import (
	"net/http"

	"github.com/illia-malachyn/food-delivery/payment/application"
)

func RefundPaymentHandler(service *application.PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentID := r.PathValue("id")
		if err := service.Refund(r.Context(), paymentID); err != nil {
			writeDomainError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
