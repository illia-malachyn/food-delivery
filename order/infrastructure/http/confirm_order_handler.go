package http

import (
	"net/http"

	"github.com/illia-malachyn/food-delivery/order/application"
)

func ConfirmOrderHandler(orderService *application.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			http.Error(w, "order id is required", http.StatusBadRequest)
			return
		}

		if err := orderService.Confirm(r.Context(), orderID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
