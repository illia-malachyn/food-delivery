package http

import (
	"encoding/json"
	"net/http"

	"github.com/illia-malachyn/food-delivery/payment/application"
)

type markFailedRequest struct {
	Reason string `json:"reason"`
}

func MarkFailedHandler(service *application.PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentID := r.PathValue("id")

		var request markFailedRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if err := service.MarkFailed(r.Context(), paymentID, request.Reason); err != nil {
			writeDomainError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
