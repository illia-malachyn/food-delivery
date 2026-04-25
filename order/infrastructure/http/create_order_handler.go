package http

import (
	"encoding/json"
	"net/http"

	"github.com/illia-malachyn/food-delivery/order/application"
)

type CreateOrderResponse struct {
	ID string `json:"id"`
}

func CreateOrderHandler(orderService *application.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var orderDTO application.OrderDTO
		err := json.NewDecoder(r.Body).Decode(&orderDTO)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		orderID, err := orderService.Create(r.Context(), &orderDTO)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CreateOrderResponse{ID: orderID})
	}
}
