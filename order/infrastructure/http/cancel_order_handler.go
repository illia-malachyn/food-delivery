package http

import (
	"encoding/json"
	"net/http"

	"github.com/illia-malachyn/food-delivery/order/application"
)

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

func CancelOrderHandler(orderService *application.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			http.Error(w, "order id is required", http.StatusBadRequest)
			return
		}

		var request CancelOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := orderService.Cancel(r.Context(), orderID, request.Reason); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
