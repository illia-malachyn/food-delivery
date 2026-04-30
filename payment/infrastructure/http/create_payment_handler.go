package http

import (
	"encoding/json"
	"net/http"

	"github.com/illia-malachyn/food-delivery/payment/application"
)

type createPaymentResponse struct {
	ID string `json:"id"`
}

func CreatePaymentHandler(service *application.PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto application.CreatePaymentDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		paymentID, err := service.Create(r.Context(), &dto)
		if err != nil {
			writeDomainError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, createPaymentResponse{ID: paymentID})
	}
}
